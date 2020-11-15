package provider

import (
	"context"
	"errors"
	"io"

	"github.com/urfave/cli/v2"
)

// Errors to be used by the clients.
var ErrBadConnection = errors.New("Could not connect to storage client. Check your internet connection and/or provider credentials.")

// Provider encompasesses its Client and CLI spec.
type Provider interface {
	NewClient(context.Context, *cli.Context) (Client, error)
	GetName() string
	GetFlags() []cli.Flag
}

// Client specifies the cloud-authenticated imgd client.
type Client interface {
	GetStateFile(ctx context.Context)
	UploadFile(ctx context.Context, filename string, media io.Reader) error
	DownloadFile(ctx context.Context, file string) ([]byte, error)
	RemoveFile(ctx context.Context)
	ListFiles(ctx context.Context)
	CreateLake(ctx context.Context) error
	RemoveLake(ctx context.Context)
	PurgeLake(context.Context)
}
