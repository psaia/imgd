package fs

import (
	"os"
	"path"
	"testing"
)

func TestHash(t *testing.T) {
	dir, _ := os.Getwd()
	files, err := DirectoryPhotos(path.Join(dir, "testdata"))
	if err != nil {
		t.Fatal(err)
	}

	hashMap := make(map[string]string)

	populateHash := func() {
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			hash, err := Hash(file)
			if err != nil {
				t.Fatal(err)
			}
			hashMap[hash] = f.Name()
		}
	}

	for i := 1; i <= 10; i++ {
		populateHash()
	}
	if len(hashMap) != len(files) {
		t.Errorf("expected just as many hashes as there are files with a consistent hash for each: %v", hashMap)
	}
}
