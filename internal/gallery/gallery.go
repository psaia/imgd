package gallery

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/psaia/imgd/internal/provider"
	"github.com/psaia/imgd/internal/state"
	"golang.org/x/sync/semaphore"
)

// IndexTplData is the struct which gets passed to RenderTemplate for the index page.
type IndexTplData struct {
	Albums []state.Album
}

// PhotoTplData is the struct which gets passed to RenderTemplate for the photo page.
type PhotoTplData struct {
	Photo    state.Photo
	Album    state.Album
	AlbumURL string
	Size     string
}

// AlbumTplData is the struct which gets passed to RenderTemplate for the album page.
type AlbumTplData struct {
	Photos   []state.Photo
	Album    state.Album
	AlbumURL string
}

// CreateIndexOptions are options
type CreateIndexOptions struct {
	St        state.State
	Client    provider.Client
	TplDir    string
	ThemeName string
}

// CreateAlbumOptions are options
type CreateAlbumOptions struct {
	St        state.State
	Client    provider.Client
	Album     state.Album
	Photo     state.Photo
	TplDir    string
	ThemeName string
}

// CreatePhotoOptions are options
type CreatePhotoOptions struct {
	Client    provider.Client
	Album     state.Album
	Photo     state.Photo
	TplDir    string
	ThemeName string
	Size      state.PhotoSizeType
}

// CreateTemplatesFromState will generate and save all template files based on the state object.
func CreateTemplatesFromState(ctx context.Context, client provider.Client, st state.State, album state.Album, tplDir, theme string) []error {
	var mu sync.Mutex
	errors := make([]error, 0)
	maxWorkers := 10
	sem := semaphore.NewWeighted(int64(maxWorkers))
	if err := CreateIndexTemplate(ctx, CreateIndexOptions{
		ThemeName: theme,
		TplDir:    tplDir,
		Client:    client,
		St:        st,
	}); err != nil {
		errors = append(errors, err)
	}
	if err := CreateAlbumTemplate(ctx, CreateAlbumOptions{
		ThemeName: theme,
		TplDir:    tplDir,
		Client:    client,
		Album:     album,
		St:        st,
	}); err != nil {
		errors = append(errors, err)
	}
	for _, hash := range album.Photos {
		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}
		go func(hash string) {
			defer sem.Release(1)
			p := st.GetPhoto(hash)
			if p != nil {
				for _, size := range state.GetPhotoSizeTypes() {
					if err := CreatePhotoTemplate(ctx, CreatePhotoOptions{
						Client:    client,
						Album:     album,
						Photo:     *p,
						Size:      size,
						ThemeName: theme,
						TplDir:    tplDir,
					}); err != nil {
						mu.Lock()
						errors = append(errors, err)
						mu.Unlock()
					}
				}
			}
		}(hash)
	}
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		fmt.Printf("Failed to acquire semaphore: %v", err)
	}
	return errors
}

// CreateIndexTemplate will upload the corresponding template file for an album.
func CreateIndexTemplate(ctx context.Context, opts CreateIndexOptions) error {
	tplDir, err := TemplateBaseDir(opts.TplDir, opts.ThemeName)
	if err != nil {
		return err
	}
	html, err := RenderIndexTemplate(tplDir, opts.Client.GetLakeBaseURL(), opts.St)
	if err != nil {
		return err
	}
	_, err = opts.Client.UploadFile(ctx, "index.html", bytes.NewReader(html))
	if err != nil {
		return err
	}
	return err
}

// CreateAlbumTemplate will upload the corresponding template file for an album.
func CreateAlbumTemplate(ctx context.Context, opts CreateAlbumOptions) error {
	tplDir, err := TemplateBaseDir(opts.TplDir, opts.ThemeName)
	if err != nil {
		return err
	}
	html, err := RenderAlbumTemplate(tplDir, opts.Client.GetLakeBaseURL(), opts.Album, opts.St)
	if err != nil {
		return err
	}
	_, err = opts.Client.UploadFile(ctx, opts.Album.PublicSlug(), bytes.NewReader(html))
	if err != nil {
		return err
	}
	return err
}

// CreatePhotoTemplate will upload the corresponding template for a photo.
func CreatePhotoTemplate(ctx context.Context, opts CreatePhotoOptions) error {
	tplDir, err := TemplateBaseDir(opts.TplDir, opts.ThemeName)
	if err != nil {
		return err
	}
	html, err := RenderPhotoTemplate(tplDir, opts.ThemeName, opts.Album, opts.Photo, opts.Size)
	if err != nil {
		return err
	}
	_, err = opts.Client.UploadFile(ctx, opts.Photo.PublicSlug(opts.Album, opts.Size), bytes.NewReader(html))
	if err != nil {
		return err
	}
	return err
}

// TemplateBaseDir will return the
func TemplateBaseDir(base, theme string) (string, error) {
	if theme == "" {
		theme = "limpo"
	}
	if base == "" {
		base = "./templates"
	}
	b, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	fullPath := path.Join(b, theme)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("template directory does not exist: %s", fullPath)
	}
	return fullPath, nil
}

// RenderIndexTemplate will create the bytes.
func RenderIndexTemplate(tplDir, bucketURL string, st state.State) ([]byte, error) {
	t, err := template.New("index.tpl.html").Funcs(template.FuncMap{
		"getAlbumURL": func(album state.Album) string {
			return album.PublicURL(bucketURL)
		},
	}).ParseFiles(path.Join(tplDir, "index.tpl.html"))
	if err != nil {
		return nil, err
	}
	w := &bytes.Buffer{}
	if err := t.Execute(w, IndexTplData{
		Albums: st.Albums,
	}); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// RenderPhotoTemplate will create the bytes.
func RenderPhotoTemplate(tplDir, bucketURL string, a state.Album, p state.Photo, size state.PhotoSizeType) ([]byte, error) {
	t, err := template.New("photo.tpl.html").Funcs(renderFuncs(bucketURL, a)).ParseFiles(path.Join(tplDir, "photo.tpl.html"))
	if err != nil {
		return nil, err
	}
	w := &bytes.Buffer{}
	if err := t.Execute(w, PhotoTplData{
		Photo:    p,
		Size:     string(size),
		Album:    a,
		AlbumURL: a.PublicURL(bucketURL),
	}); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// RenderAlbumTemplate will create the bytes.
func RenderAlbumTemplate(tplDir, bucketURL string, a state.Album, st state.State) ([]byte, error) {
	t, err := template.New("album.tpl.html").Funcs(renderFuncs(bucketURL, a)).ParseFiles(path.Join(tplDir, "album.tpl.html"))
	if err != nil {
		return nil, err
	}
	photoList := make([]state.Photo, len(a.Photos))
	for idx, hash := range a.Photos {
		p := st.GetPhoto(hash)
		if p == nil {
			return nil, fmt.Errorf("no photo found for hash in photos array: %s", hash)
		}
		photoList[idx] = *p
	}
	w := &bytes.Buffer{}
	if err := t.Execute(w, AlbumTplData{
		Photos:   photoList,
		Album:    a,
		AlbumURL: a.PublicURL(bucketURL),
	}); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// renderFuncs are available to photo and album templates.
func renderFuncs(bucketURL string, album state.Album) template.FuncMap {
	return template.FuncMap{
		"getPhotoPublicURL": func(photo state.Photo, size string) string {
			return photo.PublicURL(bucketURL, album, state.PhotoSizeType(size))
		},
		"getPhotoRawURL": func(photo state.Photo, size string) string {
			return photo.PublicURLRaw(bucketURL, state.PhotoSizeType(size))
		},
	}
}
