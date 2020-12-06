package main

import (
	"context"
	"time"

	"github.com/briandowns/spinner"
	"github.com/psaia/imgd/internal/state"
	"github.com/urfave/cli/v2"
)

func albumCreate(c *cli.Context) error {
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
	var album state.Album
	exitCode := func() cli.ExitCoder {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Start()
		defer s.Stop()
		album = state.NewAlbum()
		album.Name = c.Args().Get(0)
		st = st.AddAlbum(album)
		if _, err := saveState(ctx, client, st); err != nil {
			return err
		}
		return nil
	}()
	if exitCode == nil {
		prettyLog("%s has been created", album.Name)
	}
	return exitCode
}
