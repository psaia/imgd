package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/psaia/imgd/internal/fs"
	"github.com/psaia/imgd/internal/provider"
	"github.com/psaia/imgd/internal/provider/providers/gcs"
	"github.com/psaia/imgd/internal/state"
	"github.com/urfave/cli/v2"
)

type ErrorCode int

const (
	errCodeUnauthenticated ErrorCode = iota
	errCodeNoop
	errCodeMisc
	errCodeUnknownProvider
	errCodeCorruptState
	errCodeBadConnection
)

var cliErrors = map[ErrorCode]string{
	errCodeUnauthenticated: "Could not authenticate with provider.",
	errCodeUnknownProvider: "The provider you've provided is not yet supported.",
	errCodeMisc:            "An error occurred: %s",
	errCodeCorruptState:    "Your state is corrupt. You should create a new workspace.",
	errCodeBadConnection:   "Unable to connect to storage client. Check your credentials and internet connection.",
	errCodeNoop:            "No further action taken.",
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
						Name:  "list",
						Usage: "list all albums",
						Action: func(c *cli.Context) error {
							ctx := context.Background()

							p, err := getProvider(c.String("provider"))
							if err != nil {
								return fmtErr(errCodeUnknownProvider, nil)
							}

							client, err := p.NewClient(ctx, c)
							if err != nil {
								return fmtErr(errCodeMisc, err)
							}

							st := state.NewState()
							if err := provisionState(ctx, st, client); err != nil {
								return err
							}

							if len(st.Albums) == 0 {
								prettyLog("There are no albums to list.")
								return nil
							}

							for i, a := range st.Albums {
								log.Printf(`
%d. %s
%s
								`, i+1, a.Name, a.Description)
							}
							return nil
						},
					},
					{
						Name:  "create",
						Usage: "create a new album",
						Action: func(c *cli.Context) error {
							ctx := context.Background()

							p, err := getProvider(c.String("provider"))
							if err != nil {
								return fmtErr(errCodeUnknownProvider, nil)
							}

							client, err := p.NewClient(ctx, c)
							if err != nil {
								return fmtErr(errCodeMisc, err)
							}

							st := state.NewState()
							if err := provisionState(ctx, st, client); err != nil {
								return err
							}

							for i, a := range st.Albums {
								log.Printf(`
%d. %s
%s
								`, i+1, a.Name, a.Description)
							}
							return nil
						},
					},
					{
						Name:  "sync",
						Usage: "sync all photos within a folder to an album",
						Action: func(c *cli.Context) error {
							st := state.NewState()
							if err := st.HydrateLocal(false); err != nil {
								log.Fatalln(err)
							}

							albumName := st.GetAlbum(c.Args().Get(0))
							folder := c.Args().Get(1)

							if albumName == nil {
								fmt.Println("No album exists")
							}

							files, err := fs.DirectoryPhotos(folder)
							if err != nil {
								log.Fatalln(err)
							}
							for _, file := range files {
								hash, err := fs.Hash(file)
								if err != nil {
									log.Fatal(err)
								}
								st.SavePhotoHash(hash)
							}
							if err := st.SaveLocal(); err != nil {
								log.Fatalln(err)
							}
							return nil
						},
					},
					{
						Name:  "expand",
						Usage: "list all photos within a album",
						Action: func(c *cli.Context) error {
							return nil
						},
					},
					{
						Name:  "remove",
						Usage: "remove album",
						Action: func(c *cli.Context) error {
							return nil
						},
					},
					{
						Name:  "download",
						Usage: "download an album to your computer",
						Action: func(c *cli.Context) error {
							return nil
						},
					},
					{
						Name:  "publish",
						Usage: "generate a unique URL for a public gallery",
						Action: func(c *cli.Context) error {
							return nil
						},
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
		return cli.Exit(prettyErrorSprintf(fmt.Sprintf(cliErrors[code], err)), int(code))
	}
	return cli.Exit(prettyErrorSprintf(cliErrors[code]), int(code))
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
		return nil, errors.New("invalid provider")
	}
}

func provisionState(ctx context.Context, st *state.State, client provider.Client) cli.ExitCoder {
	err := st.HydrateLocal(false)
	shouldCreateNewStatePrompt := promptui.Prompt{
		Label:     "I could not find an existing workspace. Should I create a new one in this directory? Otherwise you should run `imgd load`.",
		IsConfirm: true,
	}

	if os.IsNotExist(err) {
		if _, err := shouldCreateNewStatePrompt.Run(); err != nil {
			// Opt-out of marshaling a new state.
			return fmtErr(errCodeNoop, nil)
		}
		// Now attempt to hydrate with a brand new state file.
		if err = st.HydrateLocal(true); err != nil {
			return fmtErr(errCodeMisc, err)
		}
	} else if err != nil {
		// Another issue occurred while attempting to open from local state.
		return fmtErr(errCodeMisc, err)
	}

	// If the local state is still in mid-transaction, there's something wrong.
	if st.InProgress {
		prettyLog("It looks like an action was stopped abruptly. Down-syncing state first.")

		err := state.NewState().DownSyncRemote(ctx, client)
		if errors.Is(err, provider.ErrBadConnection) {
			return fmtErr(errCodeBadConnection, nil)
		} else if err != nil {
			// If a state cannot be found, something odd is happening.
			return fmtErr(errCodeCorruptState, nil)
		}
	}

	return nil
}

func prettyLog(f string, v ...interface{}) {
	yellow := color.New(color.FgYellow).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	lead := yellow("[imgd]: ") + blue(f)
	if _, err := fmt.Printf(lead+"\n", v...); err != nil {
		log.Fatal(err)
	}
}

func prettyErrorSprintf(f string, v ...interface{}) string {
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	lead := yellow("[imgd]: ") + red(f)
	if len(v) > 0 {
		return fmt.Sprintf(lead+"\n", v)
	}
	return fmt.Sprintf(lead + "\n")
}
