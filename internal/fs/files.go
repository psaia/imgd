package fs

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"

	"github.com/h2non/filetype"
)

var namespace = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// IsPhoto determines if a file is actually a photo.
func IsPhoto(file string) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	head := make([]byte, 261)
	if _, err := f.Read(head); err != nil && err != io.EOF {
		return false, err
	}
	return filetype.IsImage(head), nil
}

// Hash generates a unique hash for a given file.
// @TODO improve by seeking directly to bytes that need to be
// read and not reading the entire file.
func Hash(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	hash := FuzzyHash(base64.StdEncoding.EncodeToString(content))
	return SHAHash(hash), nil
}

// SHAHash generates a SHA1 hash.
func SHAHash(data []byte) string {
	u := uuid.NewSHA1(namespace, data)
	return u.String()
}

// FuzzyHash implements the image hashing algo which ultimately gets sha'd.
func FuzzyHash(s string) []byte {
	n := len(s)
	return []byte(strconv.Itoa(n) + s[:5] + s[(n/2)-15:(n/2)+15] + s[n-15:n])
}

// DirectoryPhotos lists all photos within a directory.
func DirectoryPhotos(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return []string{}, err
	}
	paths := make([]string, 0)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path, err := filepath.Abs(dir + "/" + f.Name())
		if err != nil {
			return paths, err
		}
		isPhoto, err := IsPhoto(path)
		if err != nil {
			return paths, err
		}
		if isPhoto {
			paths = append(paths, path)
		}

	}
	return paths, nil
}

// CreateDirectoryIfNew will create a new directory only if one didn't exist before it.
func CreateDirectoryIfNew(p string) (string, error) {
	fullPath, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		return "", fmt.Errorf("Directory already exists: %s", fullPath)
	}
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return "", err
	}
	return fullPath, nil
}
