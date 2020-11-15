package gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	"cloud.google.com/go/storage"
	"github.com/psaia/imgd/internal/provider"
	"google.golang.org/api/option"
)

type Client struct {
	client      *storage.Client
	projectID   string
	cacheMaxAge int
	bucketName  string
}

var _ provider.Client = &Client{}

type ClientOptions struct {
	CredFile string
}

func NewClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(opts.CredFile))
	if err != nil {
		return nil, err
	}
	extractedProjectID, err := getProjectIDFromCredentialsFile(opts.CredFile)
	if err != nil {
		return nil, err
	}
	return &Client{
		bucketName:  "imgd",
		projectID:   extractedProjectID,
		cacheMaxAge: 86400,
		client:      client,
	}, err
}

func (c *Client) UploadFile(ctx context.Context, filename string, media io.Reader) error {
	// object := &storage.Object{
	// 	Name:         name,
	// 	CacheControl: fmt.Sprintf("public, max-age=%d", c.CacheMaxAge),
	// }
	// _, err := c.client.Objects.Insert(c.Bucket, object).Media(media).Do()
	// if err != nil {
	// 	return err
	// }
	// pretty.Println(object)
	return nil
}

func (c *Client) DownloadFile(ctx context.Context, file string) ([]byte, error) {
	rc, err := c.client.Bucket(c.bucketName).Object(file).NewReader(ctx)
	if err != nil {
		if isConnectivityErr(err) {
			return nil, provider.ErrBadConnection
		}
		return nil, fmt.Errorf("Object(%q).NewReader: %v", file, err)
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
	}
	fmt.Printf("Blob %v downloaded.\n", file)
	return data, nil
}

func (c *Client) CreateLake(ctx context.Context) error {
	bkt := c.client.Bucket(c.bucketName)
	if err := bkt.Create(ctx, c.projectID, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) RemoveFile(ctx context.Context) {
}

func (c *Client) GetStateFile(ctx context.Context) {
}

func (c *Client) ListFiles(ctx context.Context) {
}

func (c *Client) PurgeLake(ctx context.Context) {
}

func (c *Client) RemoveLake(ctx context.Context) {
}

// Pull the project name from the service account credentials file.
func getProjectIDFromCredentialsFile(credFile string) (string, error) {
	jsonFile, err := os.Open(credFile)
	if err != nil {
		return "", err
	}
	defer jsonFile.Close()
	fileBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return "", err
	}
	type creds struct {
		ProjectID string `json:"project_id"`
	}
	c := &creds{}
	if err := json.Unmarshal(fileBytes, &c); err != nil {
		return "", err
	}
	return c.ProjectID, nil
}

// A quick way to check that the error is network related. In which case
// a provider.ErrBadConnection error would be sent back.
func isConnectivityErr(err error) bool {
	_, ok := err.(net.Error)
	return ok
}
