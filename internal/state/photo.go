package state

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
	"github.com/psaia/imgd/internal/fs"
)

// Photo represents a photograph.
type Photo struct {
	Name      string `json:"name"`
	Extension string `json:"ext"`
	Hash      string `json:"hash"`
}

// PhotoSizeType represents each image size.
type PhotoSizeType string

var (
	// PhotoSizeTypeOriginal is the full size image.
	PhotoSizeTypeOriginal PhotoSizeType = "original"

	// PhotoSizeTypeThumb is the thumbnail.
	PhotoSizeTypeThumb PhotoSizeType = "thumbnail"

	// PhotoSizeTypeThumbCropped is the thumbnail cropped.
	PhotoSizeTypeThumbCropped PhotoSizeType = "thumbnail-cropped"

	// PhotoSizeTypeSmall is the small image.
	PhotoSizeTypeSmall PhotoSizeType = "small"

	// PhotoSizeTypeMedium is the small image.
	PhotoSizeTypeMedium PhotoSizeType = "medium"

	// PhotoSizeTypeLarge is the small image.
	PhotoSizeTypeLarge PhotoSizeType = "large"
)

// RawFilename formats the name based on the photo size.
func (p Photo) RawFilename(size PhotoSizeType) string {
	if size == PhotoSizeTypeOriginal {
		return fmt.Sprintf("%s.%s", p.Hash, p.Extension)
	}
	return fmt.Sprintf("%s-%s.%s", p.Hash, string(size), "jpg")
}

// PublicSlug generates the html version of a file.
func (p Photo) PublicSlug(a Album, size PhotoSizeType) string {
	return fmt.Sprintf("%s/%s-%s.html", a.ID, p.Hash, string(size))
}

// PublicURL generates the html version of a file.
func (p Photo) PublicURL(bucketURL string, a Album, size PhotoSizeType) string {
	return fmt.Sprintf("%s/%s", bucketURL, p.PublicSlug(a, size))
}

// PublicURLRaw generates the url for the actual image file.
func (p Photo) PublicURLRaw(bucketURL string, size PhotoSizeType) string {
	return fmt.Sprintf("%s/%s", bucketURL, p.RawFilename(size))
}

// MarshalPhotoFromSrc will create a photo object from a src path. If the
// photo is already persisted in the state, the persisted obj will be returned.
func (s State) MarshalPhotoFromSrc(src string) (Photo, bool, error) {
	ft, err := fileType(src)
	if err != nil {
		return Photo{}, false, err
	}
	hash, err := fs.Hash(src)
	if err != nil {
		return Photo{}, false, err
	}
	if persistedPhoto := s.GetPhoto(hash); persistedPhoto != nil {
		return *persistedPhoto, true, nil
	}
	name := path.Base(src)
	return Photo{
		Name:      strings.TrimSuffix(name, filepath.Ext(name)),
		Extension: ft.Extension,
		Hash:      hash,
	}, false, nil
}

// PersistPhoto will store a new photo hash.
func (s State) PersistPhoto(photo Photo) State {
	s.Hashes[photo.Hash] = photo
	return s
}

// GetPhoto by the hash.
func (s State) GetPhoto(hash string) *Photo {
	if photo, ok := s.Hashes[hash]; ok {
		return &photo
	}
	return nil
}

// RemovePhotoSafe from the global hashmap if there are no occurrences.
func (s State) RemovePhotoSafe(photo Photo) State {
	if s.Occurrences(photo) == 0 {
		delete(s.Hashes, photo.Hash)
	}
	return s
}

// Occurrences is the number of times a photo shows up in the state.
func (s State) Occurrences(photo Photo) int {
	i := 0
	for _, a := range s.Albums {
		for _, p := range a.Photos {
			if p == photo.Hash {
				i = i + 1
			}
		}
	}
	return i
}

// GetPhotoSizeTypes returns all sizes in an array.
func GetPhotoSizeTypes() []PhotoSizeType {
	return []PhotoSizeType{
		PhotoSizeTypeOriginal,
		PhotoSizeTypeThumb,
		PhotoSizeTypeThumbCropped,
		PhotoSizeTypeSmall,
		PhotoSizeTypeMedium,
		PhotoSizeTypeLarge,
	}
}

// GetPhotoDim gets the size based on the size type.
// w, h, fit|contrain (1 | 0)
func GetPhotoDim(sizeType PhotoSizeType) []int {
	sizes := map[PhotoSizeType][]int{
		PhotoSizeTypeOriginal:     {0, 0, 0},
		PhotoSizeTypeThumb:        {250, 250, 0},
		PhotoSizeTypeThumbCropped: {250, 250, 1},
		PhotoSizeTypeSmall:        {650, 650, 0},
		PhotoSizeTypeMedium:       {1400, 1400, 0},
		PhotoSizeTypeLarge:        {3500, 3500, 0},
	}
	return sizes[sizeType]
}

// Obtain extension for file.
func fileType(file string) (types.Type, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return types.Unknown, err
	}
	return filetype.Match(buf)
}
