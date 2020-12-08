package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/psaia/imgd/internal/provider"
	"github.com/psaia/imgd/internal/provider/providers/gcs"
	"github.com/psaia/imgd/internal/state"
	"github.com/urfave/cli/v2"
)

// ErrorCode is an error code int which will be sent during an exit.
type ErrorCode int

const (
	errCodeUnauthenticated ErrorCode = iota
	errCodeNoop
	errCodeMisc
	errCodeUnknownProvider
	errCodeCorruptState
	errCodeBadConnection
	errCodeEmptyRemoteState
)

var cliErrors = map[ErrorCode]string{
	errCodeUnauthenticated:  "Could not authenticate with provider.",
	errCodeUnknownProvider:  "The provider you've provided is not yet supported.",
	errCodeMisc:             "Problem: %v",
	errCodeCorruptState:     "Your state is corrupt. You should create a new workspace.",
	errCodeBadConnection:    "Unable to connect to storage client. Check your credentials and internet connection.",
	errCodeNoop:             "Aborted",
	errCodeEmptyRemoteState: "There's a local state, but no remote state. Check to make sure you're using the correct provider account.",
}

func main() {
	providerFlags := make([]cli.Flag, 0)

	providerFlags = append(providerFlags, []cli.Flag{
		&cli.StringFlag{
			Name:     "provider",
			Usage:    fmt.Sprintf("Currently supported providers: %s.", strings.Join(activeProviders(), ", ")),
			EnvVars:  []string{"IMGD_PROVIDER"},
			Aliases:  []string{"p"},
			Required: true,
		},
	}...)

	for _, name := range activeProviders() {
		p, err := getProvider(name)
		if err != nil {
			log.Fatal(err)
		}
		providerFlags = append(providerFlags, p.GetFlags()...)
	}

	app := &cli.App{
		Name:  "imgd",
		Flags: providerFlags,
		Commands: []*cli.Command{
			{
				Name:  "album",
				Usage: "albums are collections of photos",
				Subcommands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "list all albums",
						Action: albumList,
					},
					{
						Name:  "create",
						Usage: "create a new album",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "title",
								Value:    "",
								Usage:    "Provide a title for your photo album",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "description",
								Value: "",
								Usage: "Provide a description for your photo album",
							},
						},
						Action: albumCreate,
					},
					{
						Name:   "sync",
						Usage:  "sync all photos within a folder to an album",
						Action: albumSync,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "title",
								Value: "",
								Usage: "Update the title of the photo album",
							},
							&cli.StringFlag{
								Name:  "description",
								Value: "",
								Usage: "Update the description of the photo album",
							},
						},
					},
					{
						Name:   "remove",
						Usage:  "remove album",
						Action: albumRemove,
					},
					{
						Name:   "expand",
						Usage:  "list all photos within a album",
						Action: albumExpand,
					},
					{
						Name:   "download",
						Usage:  "download an album to your computer",
						Action: albumDownload,
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func fmtErr(code ErrorCode, err error) cli.ExitCoder {
	if err != nil {
		return cli.Exit(prettyErrorStr(fmt.Sprintf(cliErrors[code], err)), int(code))
	}
	return cli.Exit(prettyErrorStr(cliErrors[code]), int(code))
}

func activeProviders() []string {
	return []string{
		gcs.Name,
		// aws.Name,
	}
}

func getProvider(name string) (provider.Provider, error) {
	switch {
	case name == gcs.Name:
		return gcs.NewProvider(), nil
	// case name == aws.Name:
	// 	return aws.NewProvider(), nil
	default:
		prettyDebug("%s is not a real provider.", name)
		return nil, errors.New("invalid provider")
	}
}

// Find the remote state if it exists. If one does not exists, returns nil, nil.
func findRemoteState(ctx context.Context, client provider.Client) (state.State, cli.ExitCoder) {
	lakeName, err := client.FindLakeName(ctx)
	if errors.Is(err, provider.ErrNotExist) {
		return state.State{}, nil
	} else if err != nil {
		return state.State{}, fmtErr(errCodeMisc, err)
	}
	client.SetLakeName(lakeName)

	remoteState, err := state.FetchRemote(ctx, client)
	if errors.Is(err, provider.ErrBadConnection) {
		return state.State{}, fmtErr(errCodeBadConnection, nil)
	} else if errors.Is(err, provider.ErrNotExist) {
		return state.State{}, nil
	} else if err != nil {
		return state.State{}, fmtErr(errCodeMisc, err)
	}
	if err := remoteState.SaveLocal(); err != nil {
		return state.State{}, fmtErr(errCodeMisc, err)
	}
	return remoteState, nil
}

func findLocalState(ctx context.Context, client provider.Client) (state.State, cli.ExitCoder) {
	localState, err := state.FetchLocal()
	if os.IsNotExist(err) {
		return state.State{}, nil
	} else if err != nil {
		return state.State{}, fmtErr(errCodeMisc, err)
	}
	client.SetLakeName(localState.LakeName)
	return localState, nil
}

func createNewState(ctx context.Context, client provider.Client) (state.State, cli.ExitCoder) {
	st := state.New()
	client.SetLakeName(st.LakeName)
	err := client.CreateLake(ctx)
	if err != nil {
		prettyDebug("Error while creating new lake.")
		return state.State{}, fmtErr(errCodeMisc, err)
	}
	return saveState(ctx, client, st)
}

func saveState(ctx context.Context, client provider.Client, st state.State) (state.State, cli.ExitCoder) {
	if err := st.SaveLocal(); err != nil {
		prettyDebug("Error while saving state locally.")
		return state.State{}, fmtErr(errCodeMisc, err)
	}
	if err := st.SaveRemote(ctx, client); err != nil {
		prettyDebug("Error while saving state remotely.")
		return state.State{}, fmtErr(errCodeMisc, err)
	}
	return st, nil
}

func provisionState(ctx context.Context, client provider.Client) (state.State, cli.ExitCoder) {
	localState, err := findLocalState(ctx, client)
	if err != nil {
		prettyDebug("Error occurred while finding local state.")
		return state.State{}, err
	}
	if !isEmptyState(localState) {
		prettyDebug("Found local state file.")
		return localState, nil
	}
	remoteState, err := findRemoteState(ctx, client)
	if err != nil {
		prettyDebug("Error occurred while finding remote state.")
		return state.State{}, err
	}
	if !isEmptyState(remoteState) {
		prettyDebug("Refreshed from remote state.")
		return remoteState, nil
	}
	prettyDebug("Could not detect a local or remote state so provisioning a new workspace.")
	newState, err := createNewState(ctx, client)
	if err != nil {
		prettyDebug("Error occurred while creating new state.")
		return state.State{}, err
	}
	return newState, nil
}

func isEmptyState(st state.State) bool {
	return st.ID == ""
}

func processingConcurrency() int {
	concurrency := runtime.NumCPU()
	if os.Getenv("CONCURRENCY") != "" {
		if i, err := strconv.Atoi(os.Getenv("CONCURRENCY")); err == nil {
			return i
		}
	}
	return concurrency
}
