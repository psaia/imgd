package fs

import (
	"bufio"
	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"

	"github.com/h2non/filetype"
)

var namespace = uuid.MustParse("00000000-0000-0000-0000-000000000000")

func IsPhoto(file string) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer f.Close()
	head := make([]byte, 261)
	if _, err := f.Read(head); err != nil && err != io.EOF {
		return false, err
	}
	return filetype.IsImage(head), nil
}

func Hash(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	hash := FuzzyHash(base64.StdEncoding.EncodeToString(content))
	return SHAHash(hash), nil
}

func SHAHash(data []byte) string {
	u := uuid.NewSHA1(namespace, data)
	return u.String()
}

func FuzzyHash(s string) []byte {
	n := len(s)
	return []byte(strconv.Itoa(n) + s[:5] + s[(n/2)-15:(n/2)+15] + s[n-15:n])
}

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
