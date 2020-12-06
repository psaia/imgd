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
	folder := c.Args().Get(1)
	files, err := fs.DirectoryPhotos(folder)
	if err != nil {
		return fmtErr(errCodeMisc, err)
	}
	creating, removing, err := syncPrep(files, st)
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
		prettyError("Encountered error during removal: %s", err)
	}
	if exitCode == nil {
		prettyLog("%s has been synced", album.Name)
	}
	return exitCode
}

func syncResizeTask(ctx context.Context, errc chan<- error, stream <-chan albumSyncJob) <-chan albumSyncJob {
	out := make(chan albumSyncJob)
	go func() {
		defer close(out)
		for job := range stream {
			if job.size != state.PhotoSizeTypeOriginal {
				dir, err := ioutil.TempDir("", "imgd-imgcache")
				if err != nil {
					errc <- err
					continue
				}
				job.dstFilePath = fmt.Sprintf("%s/%s", dir, job.photo.FormattedFileName(job.size))
				src, err := imaging.Open(job.srcFilePath, imaging.AutoOrientation(true))
				if err != nil {
					errc <- err
					continue
				}
				dim := state.GetPhotoDim(job.size)
				var dst *image.NRGBA
				if dim[2] == 1 {
					dst = imaging.Fill(src, dim[0], dim[1], imaging.Center, imaging.Lanczos)
				} else {
					dst = imaging.Fit(src, dim[0], dim[1], imaging.Lanczos)
				}
				if err = imaging.Save(dst, job.dstFilePath); err != nil {
					errc <- err
					continue
				}
			} else {
				job.dstFilePath = job.srcFilePath
			}
			select {
			case <-ctx.Done():
				prettyDebug("upload pipeline task breaking because expired")
				break
			case out <- job:
			}
		}
	}()
	return out
}

func syncUploadTask(ctx context.Context, errc chan<- error, client provider.Client, stream <-chan albumSyncJob) <-chan albumSyncJob {
	out := make(chan albumSyncJob)
	go func() {
		defer close(out)
		for job := range stream {
			r, err := os.Open(job.dstFilePath)
			if err != nil {
				errc <- err
				continue
			}
			url, err := client.UploadFile(ctx, job.photo.FormattedFileName(job.size), r)
			if err != nil {
				errc <- err
				continue
			}
			job.photo.URL = url
			select {
			case <-ctx.Done():
				prettyDebug("upload pipeline task breaking because expired")
				break
			case out <- job:
			}
		}
	}()
	return out
}

func syncCleanupTask(ctx context.Context, errc chan<- error, stream <-chan albumSyncJob) <-chan albumSyncJob {
	out := make(chan albumSyncJob)
	go func() {
		defer close(out)
		for job := range stream {
			if job.size != state.PhotoSizeTypeOriginal { // Only remove an image if it has a custom w/h. Otherwise, it's the original.
				if err := os.RemoveAll(path.Dir(job.dstFilePath)); err != nil {
					errc <- err
					continue
				}
			}
			select {
			case <-ctx.Done():
				prettyDebug("upload pipeline task breaking because expired")
				break
			case out <- job:
			}
		}
	}()
	return out
}

func syncPrep(files []string, st state.State) ([]albumSyncJob, []albumSyncJob, error) {
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
	for hash := range st.Hashes {
		if photo, exists := preExistingHash[hash]; !exists {
			for _, size := range state.GetPhotoSizeTypes() {
				forRemoval = append(forRemoval, albumSyncJob{
					photo:  photo,
					remove: true,
					size:   size,
				})
			}
		}
	}
	return forCreation, forRemoval, nil
}

func syncPrompt(forCreation, forRemoval []albumSyncJob) bool {
	addList := "Nothing to add."
	removeList := "Nothing to remove."
	for _, job := range forCreation {
		if job.size == state.PhotoSizeTypeOriginal {
			addList = fmt.Sprintf("%s+ %s [%s]\n", addList, job.photo.Hash, job.photo.Name)
		}
	}
	for _, job := range forRemoval {
		if job.size == state.PhotoSizeTypeOriginal {
			removeList = fmt.Sprintf("%s- %s [%s]\n", removeList, job.photo.Hash, job.photo.Name)
		}
	}
	prettyLog("\nAdding:\n%s\n\nRemoving:\n%s\n", addList, removeList)
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
	maxWorkers := 20
	sem := semaphore.NewWeighted(int64(maxWorkers))

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
			jobStream := make(chan albumSyncJob, 1)
			jobStream <- j
			stream := syncCleanupTask(
				ctx,
				errc,
				syncUploadTask(
					ctx,
					errc,
					client,
					syncResizeTask(ctx, errc, jobStream),
				))
			select {
			case job := <-stream:
				// Only the original size photos need to be persisted.
				if job.size == state.PhotoSizeTypeOriginal {
					mu.Lock()
					st = st.PersistPhoto(job.photo)
					st = st.AddPhotoToAlbum(album, job.photo)
					mu.Unlock()
				}
				break
			}
		}(j)
	}
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		prettyDebug("Failed to acquire semaphore: %v", err)
	}

	for _, job := range forRemoval {
		if err := client.RemoveFile(ctx, job.photo.FormattedFileName(job.size)); err != nil {
			errors = append(errors, fmt.Errorf("Encountered error while removing photo from storage: %v", err))
			continue
		}
		if job.size == state.PhotoSizeTypeOriginal {
			st = st.RemovePhotoFromAlbum(album, job.photo)
			st = st.RemovePhotoSafe(job.photo)
		}
	}
	close(errc)
	wg.Wait()
	return st, errors
}