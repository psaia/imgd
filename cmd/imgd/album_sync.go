package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/disintegration/imaging"
	"github.com/manifoldco/promptui"
	"github.com/psaia/imgd/internal/fs"
	"github.com/psaia/imgd/internal/gallery"
	"github.com/psaia/imgd/internal/provider"
	"github.com/psaia/imgd/internal/state"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"
)

// albumSyncJob represents a media item which will be added or removed.
type albumSyncJob struct {
	srcFilePath string
	size        state.PhotoSizeType
	photo       state.Photo
	dstFilePath string
	remove      bool
}

func albumSync(c *cli.Context) error {
	ctx := context.Background()
	p, err := getProvider(c.String("provider"))
	if err != nil {
		return fmtErr(errCodeUnknownProvider, nil)
	}
	client, err := p.NewClient(ctx, c)
	if err != nil {
		return fmtErr(errCodeMisc, err)
	}
	st, err := provisionState(ctx, client)
	if err != nil {
		return err
	}
	album := st.GetAlbum(c.Args().Get(0))
	if album == nil {
		return fmtErr(errCodeMisc, errors.New("Album does not exist"))
	}
	if c.String("title") != "" {
		album.Name = c.String("title")
	}
	if c.String("description") != "" {
		album.Description = c.String("description")
	}
	folder := c.Args().Get(1)
	files, err := fs.DirectoryPhotos(folder)
	if err != nil {
		return fmtErr(errCodeMisc, err)
	}
	creating, removing, err := syncPrep(files, st, *album)
	if err != nil {
		return fmtErr(errCodeMisc, err)
	}
	if confirmed := syncPrompt(creating, removing); !confirmed {
		return fmtErr(errCodeNoop, nil)
	}
	var errs []error
	exitCode := func() cli.ExitCoder {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Start()
		defer s.Stop()
		st, errs = syncRun(ctx, client, *album, st, creating, removing)
		if _, err := saveState(ctx, client, st); err != nil {
			return err
		}
		return nil
	}()
	for _, err := range errs {
		prettyError("Encountered error during sync: %s", err)
	}
	if exitCode == nil {
		prettyLog("%s has been synced", album.Name)
	}
	return exitCode
}

func syncResizeTask(ctx context.Context, job *albumSyncJob) error {
	if job.size != state.PhotoSizeTypeOriginal {
		dir, err := ioutil.TempDir("", "imgd-imgcache")
		if err != nil {
			return err
		}
		prettyDebug("%s: Resizing started", job.photo.RawFilename(job.size))
		job.dstFilePath = fmt.Sprintf("%s/%s", dir, job.photo.RawFilename(job.size))
		src, err := imaging.Open(job.srcFilePath, imaging.AutoOrientation(true))
		if err != nil {
			prettyDebug("Error occurred while opening src file to be resized (%s): %v", job.srcFilePath, err)
			return err
		}
		dim := state.GetPhotoDim(job.size)
		var dst *image.NRGBA
		if dim[2] == 1 {
			dst = imaging.Fill(src, dim[0], dim[1], imaging.Center, imaging.Lanczos)
		} else {
			dst = imaging.Fit(src, dim[0], dim[1], imaging.Lanczos)
		}
		if err = imaging.Save(dst, job.dstFilePath); err != nil {
			prettyDebug("Error occurred while saving resized photo (%s): %v", job.dstFilePath, err)
			return err
		}
		prettyDebug("%s: Resizing completed", job.photo.RawFilename(job.size))
	} else {
		// No need to ressize for original photos. Just point the dst to the src.
		job.dstFilePath = job.srcFilePath
	}
	return nil
}

func syncUploadTask(ctx context.Context, client provider.Client, job *albumSyncJob) error {
	r, err := os.Open(job.dstFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			prettyError("Encountered error while trying to close file: %v", err)
		}
	}()
	prettyDebug("%s: Uploading started", job.photo.RawFilename(job.size))
	_, err = client.UploadFile(ctx, job.photo.RawFilename(job.size), r)
	if err != nil {
		prettyDebug("Error occurred while uploading to storage: %v", err)
		return err
	}
	prettyDebug("%s: Uploading completed", job.photo.RawFilename(job.size))
	return nil
}

func syncCleanupTask(ctx context.Context, job *albumSyncJob) error {
	if job.size != state.PhotoSizeTypeOriginal { // Only remove an image if it has a custom w/h. Otherwise, it's the original.
		prettyDebug("%s: Removing", job.dstFilePath)
		if err := os.RemoveAll(path.Dir(job.dstFilePath)); err != nil {
			return err
		}
	}
	return nil
}

func syncPrep(files []string, st state.State, a state.Album) ([]albumSyncJob, []albumSyncJob, error) {
	forRemoval := make([]albumSyncJob, 0)
	forCreation := make([]albumSyncJob, 0)
	preExistingHash := make(map[string]state.Photo)

	for _, file := range files {
		photo, exists, err := st.MarshalPhotoFromSrc(file)
		if err != nil {
			return forCreation, forRemoval, err
		}
		preExistingHash[photo.Hash] = photo
		if !exists {
			for _, size := range state.GetPhotoSizeTypes() {
				forCreation = append(forCreation, albumSyncJob{
					srcFilePath: file,
					size:        size,
					photo:       photo,
				})
			}
		}
	}
	for _, hash := range a.Photos {
		if _, exists := preExistingHash[hash]; !exists {
			for _, size := range state.GetPhotoSizeTypes() {
				photo := st.GetPhoto(hash)
				forRemoval = append(forRemoval, albumSyncJob{
					photo:  *photo,
					remove: true,
					size:   size,
				})
			}
		}
	}
	return forCreation, forRemoval, nil
}

func syncPrompt(forCreation, forRemoval []albumSyncJob) bool {
	var addList, removeList string

	if len(forCreation) > 0 {
		for _, job := range forCreation {
			if job.size == state.PhotoSizeTypeOriginal {
				addList = fmt.Sprintf("%s+ %s [%s]\n", addList, job.photo.Hash, job.photo.Name)
			}
		}
	} else {
		addList = "Nothing to add."
	}
	if len(forRemoval) > 0 {
		for _, job := range forRemoval {
			if job.size == state.PhotoSizeTypeOriginal {
				removeList = fmt.Sprintf("%s- %s [%s]\n", removeList, job.photo.Hash, job.photo.Name)
			}
		}
	} else {
		removeList = "Nothing to remove."
	}
	prettyLog("\nAdding:\n%s\n\nRemoving:\n%s\n", addList, removeList)
	if len(forCreation) == 0 && len(forRemoval) == 0 {
		fmt.Println(prettyLogStr("There are not updates but if you proceed all html templates will regenerate."))
	}
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("Are you sure you would like to proceed"),
		IsConfirm: true,
	}
	str, _ := prompt.Run()
	return str == "y"
}

func syncRun(ctx context.Context, client provider.Client, album state.Album, st state.State, forCreation, forRemoval []albumSyncJob) (state.State, []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)
	errc := make(chan error)
	maxWorkers := processingConcurrency()
	sem := semaphore.NewWeighted(int64(maxWorkers))
	prettyDebug("Sync concurrency set to %d", maxWorkers)

	wg.Add(1) // Resolves when errors chan is closed.

	go func() {
		defer wg.Done()
		for err := range errc {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}
	}()

	for _, j := range forCreation {
		if err := sem.Acquire(ctx, 1); err != nil {
			prettyDebug("Failed to acquire semaphore: %v", err)
			break
		}
		go func(j albumSyncJob) {
			defer sem.Release(1)
			job := &j
			if err := syncResizeTask(ctx, job); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
			if err := syncUploadTask(ctx, client, job); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			} else {
				// Only the original size photos need to be persisted.
				if job.size == state.PhotoSizeTypeOriginal {
					mu.Lock()
					st = st.PersistPhoto(job.photo)
					st = st.AddPhotoToAlbum(album, job.photo)
					mu.Unlock()
				}
			}
			if err := syncCleanupTask(ctx, job); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(j)
	}
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		prettyDebug("Failed to acquire semaphore: %v", err)
	}
	for _, job := range forRemoval {
		if err := client.RemoveFile(ctx, job.photo.RawFilename(job.size)); err != nil {
			errors = append(errors, fmt.Errorf("Encountered error while removing photo from storage: %v", err))
		}
		if err := client.RemoveFile(ctx, job.photo.PublicSlug(album, job.size)); err != nil {
			errors = append(errors, fmt.Errorf("Encountered error while removing photo HTML template from storage: %v", err))
		}
		if job.size == state.PhotoSizeTypeOriginal {
			st = st.RemovePhotoFromAlbum(album, job.photo)
			st = st.RemovePhotoSafe(job.photo)
		}
		prettyDebug("Removed photo: %s", job.photo.Name)
	}
	album = *(st.GetAlbum(album.ID))
	if errs := gallery.CreateTemplatesFromState(ctx, client, st, album, "", ""); len(errs) > 0 {
		for _, e := range errs {
			errc <- e
		}
	}
	close(errc)
	wg.Wait()
	return st, errors
}
