package state

import (
	"testing"
)

func TestAddAlbum(t *testing.T) {
	st := New()
	album := NewAlbum()
	st = st.AddAlbum(album)
	if len(st.Albums) != 1 {
		t.Fatalf("expected 1 album. got %d", len(st.Albums))
	}
}

func TestAddPhotoToAlbum(t *testing.T) {
	st := New()
	album := NewAlbum()
	st = st.AddAlbum(album)
	st = st.AddPhotoToAlbum(album, Photo{Hash: "abc"})
	st = st.AddPhotoToAlbum(album, Photo{Hash: "efg"})
	if len(st.Albums[0].Photos) != 2 {
		t.Fatalf("expected the first album to contain to photos. got %d", len(st.Albums[0].Photos))
	}
}

func TestRemovePhotoToAlbum(t *testing.T) {
	st := New()
	album := NewAlbum()
	st = st.AddAlbum(album)
	st = st.AddPhotoToAlbum(album, Photo{Hash: "abc"})
	st = st.AddPhotoToAlbum(album, Photo{Hash: "efg"})
	st = st.RemovePhotoFromAlbum(album, Photo{Hash: "efg"})
	if len(st.Albums[0].Photos) != 1 {
		t.Fatalf("expected 1 but got %d", len(st.Albums[0].Photos))
	}
	st = st.RemovePhotoFromAlbum(album, Photo{Hash: "abc"})
	if len(st.Albums[0].Photos) != 0 {
		t.Fatalf("expected 0 but got %d", len(st.Albums[0].Photos))
	}
}

func TestRemoveAlbum(t *testing.T) {
	st := New()
	album := NewAlbum()
	st = st.AddAlbum(album)
	st = st.RemoveAlbum(album)
	if len(st.Albums) != 0 {
		t.Fatalf("expected the album to be removed but it wasn't")
	}
}
