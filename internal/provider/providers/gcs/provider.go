package gcs

import (
	"context"

	"github.com/psaia/imgd/internal/provider"
	"github.com/urfave/cli/v2"
)

// Name of provider.
const Name = "gcs"

// Instructions for provider.
const Instructions = `
Alternatively, set the environmental variable 'IMGD_GCS_CREDENTIALS'.

Path to a GCP JSON service account key file.

You can create a service account file here: https://console.cloud.google.com/iam-admin/serviceaccounts?_ga=2.247386500.1834966576.1605452184-1907820296.1605452184
`

// Provider struct.
type Provider struct{}

var _ provider.Provider = Provider{}

func NewProvider() Provider {
	return Provider{}
}

func (Provider) NewClient(ctx context.Context, cliCtx *cli.Context) (provider.Client, error) {
	opts := ClientOptions{
		CredFile: cliCtx.String("gcs-creds"),
	}
	return New(ctx, opts)
}

func (Provider) GetName() string {
	return Name
}

func (Provider) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "gcs-creds",
			Usage:    Instructions,
			EnvVars:  []string{"IMGD_GCS_CREDENTIALS"},
			Required: false,
		},
	}
}
