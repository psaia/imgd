package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
)

func albumList(c *cli.Context) error {
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
	if len(st.Albums) == 0 {
		prettyLog("There are no albums to list.")
		return nil
	}
	prettyLog(fmt.Sprintf("All albums:\n\n"))
	for i, a := range st.Albums {
		prettyLog(fmt.Sprintf("%d. %s (%d)\n  [%s]", i+1, a.Name, len(a.Photos), a.Description))
	}
	return nil
}
