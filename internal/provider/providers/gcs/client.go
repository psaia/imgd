package gcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"regexp"

	"cloud.google.com/go/storage"
	"github.com/psaia/imgd/internal/provider"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Client which implements provider.Client
type Client struct {
	client      *storage.Client
	projectID   string
	cacheMaxAge int
	lakeName    string
}

var _ provider.Client = &Client{}

// ClientOptions for a NewClient
type ClientOptions struct {
	CredFile string
}

// New provisions a new Client
func New(ctx context.Context, opts ClientOptions) (*Client, error) {
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(opts.CredFile))
	if err != nil {
		return nil, err
	}
	extractedProjectID, err := getProjectIDFromCredentialsFile(opts.CredFile)
	if err != nil {
		return nil, err
	}
	return &Client{
		projectID:   extractedProjectID,
		cacheMaxAge: 86400,
		client:      client,
	}, err
}

// UploadFile will upload a file to the bucket.
func (c *Client) UploadFile(ctx context.Context, filename string, media io.Reader) (string, error) {
	wc := c.client.Bucket(c.GetLakeName()).Object(filename).NewWriter(ctx)
	if _, err := io.Copy(wc, media); err != nil {
		return "", fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %v", err)
	}
	return imgURL(c.lakeName, path.Base(filename)), nil
}

// DownloadFile will download a specific file by its name.
func (c *Client) DownloadFile(ctx context.Context, file string) ([]byte, error) {
	rc, err := c.client.Bucket(c.GetLakeName()).Object(file).NewReader(ctx)
	if err != nil {
		if isConnectivityErr(err) {
			return nil, provider.ErrBadConnection
		} else if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, provider.ErrNotExist
		}
		return nil, fmt.Errorf("GCP.client: Object(%q).NewReader: %v", file, err)
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
	}
	return data, nil
}

// CreateLake will create a new lake given the bucket name.
func (c *Client) CreateLake(ctx context.Context) error {
	bkt := c.client.Bucket(c.GetLakeName())
	if err := bkt.Create(ctx, c.projectID, &storage.BucketAttrs{
		StorageClass:               "STANDARD",
		PredefinedDefaultObjectACL: "publicRead",
	}); err != nil {
		return err
	}
	return nil
}

// FindLakeName will return the first lakename.
func (c *Client) FindLakeName(ctx context.Context) (string, error) {
	r := regexp.MustCompile(fmt.Sprintf("^%s-.+$", provider.LakePrefix))
	it := c.client.Buckets(ctx, c.projectID)
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		if r.Match([]byte(obj.Name)) {
			return obj.Name, nil
		}
		return obj.Name, nil
	}
	return "", provider.ErrNotExist
}

// RemoveFile from the bucket.
func (c *Client) RemoveFile(ctx context.Context, filename string) error {
	if err := c.client.Bucket(c.GetLakeName()).Object(filename).Delete(ctx); err != nil {
		if isConnectivityErr(err) {
			return provider.ErrBadConnection
		} else if errors.Is(err, storage.ErrObjectNotExist) {
			return provider.ErrNotExist
		}
		return err
	}
	return nil
}

// RemoveLake will completely remove a lake.
func (c *Client) RemoveLake(ctx context.Context) {
}

// GetLakeName gets a lakeName
func (c *Client) GetLakeName() string {
	return c.lakeName
}

// SetLakeName sets a new lakeName
func (c *Client) SetLakeName(name string) {
	c.lakeName = name
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

func imgURL(bucketName string, filename string) string {
	return fmt.Sprintf("http://%s.storage.googleapis.com/%s", bucketName, filename)
}
