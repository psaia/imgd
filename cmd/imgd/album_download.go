package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/psaia/imgd/internal/fs"
	"github.com/psaia/imgd/internal/provider"
	"github.com/psaia/imgd/internal/state"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"
)

// albumRemoveJob represents a media item which will be added or removed.
type albumDownloadJob struct {
	photo    state.Photo
	filename string
	dstPath  string
}

func albumDownload(c *cli.Context) error {
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
	if album == nil {
		return fmtErr(errCodeMisc, err)
	}
	dirPath, err := fs.CreateDirectoryIfNew(c.Args().Get(1))
	jobs := albumDownloadJobs(st, *album, dirPath)
	var errs []error
	exitCode := func() cli.ExitCoder {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Start()
		defer s.Stop()
		st, errs = albumDownloadRun(ctx, *album, st, client, jobs)
		if _, err := saveState(ctx, client, st); err != nil {
			return err
		}
		return nil
	}()
	for _, err := range errs {
		prettyError("Encountered error during removal: %s", err)
	}
	return exitCode
}

func albumDownloadJobs(st state.State, album state.Album, dirPath string) []albumDownloadJob {
	jobs := make([]albumDownloadJob, 0)
	for _, hash := range album.Photos {
		photo := st.GetPhoto(hash)
		if photo != nil {
			jobs = append(jobs, albumDownloadJob{
				photo:    *photo,
				filename: photo.RawFilename(state.PhotoSizeTypeOriginal),
				dstPath:  filepath.Join(dirPath, fmt.Sprintf("%s.%s", photo.Name, photo.Extension)),
			})
		}
	}
	return jobs
}

func albumDownloadRun(ctx context.Context, album state.Album, st state.State, client provider.Client, jobs []albumDownloadJob) (state.State, []error) {
	var mu sync.Mutex
	errors := make([]error, 0)
	maxWorkers := 20
	sem := semaphore.NewWeighted(int64(maxWorkers))

	for _, job := range jobs {
		if err := sem.Acquire(ctx, 1); err != nil {
			prettyDebug("Failed to acquire semaphore: %v", err)
			break
		}
		go func(j albumDownloadJob) {
			defer sem.Release(1)
			bytes, err := client.DownloadFile(ctx, j.filename)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
			if err := ioutil.WriteFile(j.dstPath, bytes, 0755); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(job)
	}
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		prettyDebug("Failed to acquire semaphore: %v", err)
	}
	return st, errors
}
