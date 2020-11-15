package state

import (
	"time"

	"github.com/google/uuid"
)

type Album struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Created     string  `json:"created"`
	Updated     string  `json:"updated"`
	Photos      []Photo `json:"photos"`
}

func NewAlbum() Album {
	return Album{
		Created: time.Now().String(),
		ID:      uuid.New().String(),
	}
}

func (s *State) AddAlbum(a Album) {
	s.mu.Lock()
	s.Albums = append(s.Albums, a)
	s.mu.Unlock()
}

func (s *State) GetAlbum(name string) *Album {
	for _, album := range s.Albums {
		if album.Name == name || album.ID == name {
			return &album
		}
	}
	return nil
}
