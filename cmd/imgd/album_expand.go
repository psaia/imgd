package main

import (
	"context"
	"errors"

	"github.com/urfave/cli/v2"
)

func albumExpand(c *cli.Context) error {
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
	prettyLog("\nAlbum: %s\nDescription: %s\n", album.Name, album.Description)
	for _, hash := range album.Photos {
		photo := st.GetPhoto(hash)
		if photo != nil {
			prettyLogMinimal("Name: %s\nID: %s\nURL: %s\n", photo.Name, photo.Hash, photo.URL)
		}
	}
	return nil
}
