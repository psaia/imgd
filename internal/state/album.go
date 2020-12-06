package state

import (
	"time"

	"github.com/google/uuid"
)

// Album represents a collection of photos.
type Album struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Photos      []string `json:"photos"`
}

// NewAlbum creates a new album.
func NewAlbum() Album {
	return Album{
		Created: time.Now().String(),
		ID:      uuid.New().String(),
		Photos:  make([]string, 0),
	}
}

// AddAlbum adds a new album.
func (s State) AddAlbum(a Album) State {
	s.Albums = append(s.Albums, a)
	return s
}

// GetAlbum will return an album if it exists.
func (s State) GetAlbum(name string) *Album {
	for _, album := range s.Albums {
		if album.Name == name || album.ID == name {
			return &album
		}
	}
	return nil
}

// RemoveAlbum will completely remove an album from the state.
func (s State) RemoveAlbum(a Album) State {
	for idx, album := range s.Albums {
		if album.ID == a.ID {
			s.Albums = append(s.Albums[:idx], s.Albums[idx+1:]...)
		}
	}
	return s
}

// AddPhotoToAlbum adds a photo to an album.
func (s State) AddPhotoToAlbum(a Album, p Photo) State {
	for _, hash := range a.Photos {
		if hash == p.Hash {
			return s
		}
	}
	for idx := range s.Albums {
		if s.Albums[idx].ID == a.ID {
			s.Albums[idx].Photos = append(s.Albums[idx].Photos, p.Hash)
		}
	}
	return s
}

// RemovePhotoFromAlbum adds a photo from a specific album.
func (s State) RemovePhotoFromAlbum(a Album, p Photo) State {
	for aIdx := range s.Albums {
		if a.ID == s.Albums[aIdx].ID {
			for pIdx := range s.Albums[aIdx].Photos {
				if s.Albums[aIdx].Photos[pIdx] == p.Hash {
					s.Albums[aIdx].Photos = append(s.Albums[aIdx].Photos[:pIdx], s.Albums[aIdx].Photos[pIdx+1:]...)
					return s
				}
			}
		}
	}
	return s
}
