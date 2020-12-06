package provider

import (
	"context"
	"errors"
	"io"

	"github.com/urfave/cli/v2"
)

// LakePrefix for each imgd bucket created.
const LakePrefix = "imgd"

// Errors to be used by the clients.
var ErrBadConnection = errors.New("Could not connect to storage client. Check your internet connection and/or provider credentials.")
var ErrNotExist = errors.New("The resource does not exist.")

// Provider encompasesses its Client and CLI spec.
type Provider interface {
	NewClient(context.Context, *cli.Context) (Client, error)
	GetName() string
	GetFlags() []cli.Flag
}

// Client specifies the cloud-authenticated imgd client.
type Client interface {
	FindLakeName(ctx context.Context) (string, error)
	GetLakeName() string
	SetLakeName(name string)
	UploadFile(ctx context.Context, file string, media io.Reader) (string, error)
	DownloadFile(ctx context.Context, file string) ([]byte, error)
	RemoveFile(ctx context.Context, file string) error
	CreateLake(ctx context.Context) error
	RemoveLake(ctx context.Context)
}
