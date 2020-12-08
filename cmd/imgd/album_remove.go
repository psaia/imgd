package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/manifoldco/promptui"
	"github.com/psaia/imgd/internal/gallery"
	"github.com/psaia/imgd/internal/provider"
	"github.com/psaia/imgd/internal/state"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"
)

// albumRemoveJob represents a media item which will be added or removed.
type albumRemoveJob struct {
	size  state.PhotoSizeType
	photo state.Photo
}

func albumRemove(c *cli.Context) error {
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
	jobs := albumRemoveJobs(st, *album)
	if !albumRemovePrompt(jobs) {
		return fmtErr(errCodeNoop, nil)
	}
	var errs []error
	exitCode := func() cli.ExitCoder {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Start()
		defer s.Stop()
		st, errs = albumRemoveRun(ctx, *album, st, client, jobs)
		album = st.GetAlbum(album.ID)
		if len(album.Photos) == 0 {
			st = st.RemoveAlbum(*album)
		} else {
			return fmtErr(errCodeMisc, errors.New("Could not fully remove album because there were issues removing some of the photos within it. Please try again"))
		}
		if _, err := saveState(ctx, client, st); err != nil {
			return err
		}
		return nil
	}()
	for _, err := range errs {
		prettyError("Encountered error during removal: %s", err)
	}
	if exitCode == nil {
		prettyLog("%s has been removed", album.Name)
	}
	return exitCode
}

func albumRemoveJobs(st state.State, album state.Album) []albumRemoveJob {
	jobs := make([]albumRemoveJob, 0)
	for _, hash := range album.Photos {
		photo := st.GetPhoto(hash)
		if photo != nil {
			for _, size := range state.GetPhotoSizeTypes() {
				jobs = append(jobs, albumRemoveJob{
					size:  size,
					photo: *photo,
				})
			}
		}
	}
	return jobs
}

func albumRemovePrompt(forRemoval []albumRemoveJob) bool {
	var removeList string
	if len(forRemoval) > 0 {
		for _, job := range forRemoval {
			if job.size == state.PhotoSizeTypeOriginal {
				removeList = fmt.Sprintf("%s- %s [%s]\n", removeList, job.photo.Hash, job.photo.Name)
			}
		}
	} else {
		removeList = "Nothing to remove."
	}
	prettyLog("Removing:\n%s\n", removeList)
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("Are you sure you would like to proceed"),
		IsConfirm: true,
	}
	str, _ := prompt.Run()
	return str == "y"
}

func albumRemoveRun(ctx context.Context, album state.Album, st state.State, client provider.Client, jobs []albumRemoveJob) (state.State, []error) {
	var mu sync.Mutex
	errors := make([]error, 0)
	maxWorkers := 20
	sem := semaphore.NewWeighted(int64(maxWorkers))

	for _, job := range jobs {
		if err := sem.Acquire(ctx, 1); err != nil {
			prettyDebug("Failed to acquire semaphore: %v", err)
			break
		}
		go func(j albumRemoveJob) {
			defer sem.Release(1)
			if err := client.RemoveFile(ctx, j.photo.RawFilename(j.size)); err != nil && err != provider.ErrNotExist {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
			if err := client.RemoveFile(ctx, j.photo.PublicSlug(album, j.size)); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
			if j.size == state.PhotoSizeTypeOriginal {
				mu.Lock()
				st = st.RemovePhotoFromAlbum(album, j.photo)
				st = st.RemovePhotoSafe(j.photo)
				mu.Unlock()
			}
		}(job)
	}
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		prettyDebug("Failed to acquire semaphore: %v", err)
	}
	if err := client.RemoveFile(ctx, album.PublicSlug()); err != nil {
		errors = append(errors, err)
	}
	// Regenerate the index file.
	if err := gallery.CreateIndexTemplate(ctx, gallery.CreateIndexOptions{
		ThemeName: "",
		TplDir:    "",
		Client:    client,
		St:        st,
	}); err != nil {
		errors = append(errors, err)
	}
	return st, errors
}
