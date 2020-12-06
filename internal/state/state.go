package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
	"github.com/psaia/imgd/internal/provider"
	"golang.org/x/net/context"
)

// State represents an immutable state object.
type State struct {
	ID       string           `json:"id"`
	LakeName string           `json:"lakeName"`
	Hashes   map[string]Photo `json:"_ph"`
	Albums   []Album          `json:"albums"`
}

// StateFile declares where the statefile should be saved.
const StateFile = ".imgd.state"

// New creates a new State.
func New() State {
	return State{
		ID:       uuid.New().String(),
		LakeName: fmt.Sprintf("%s-%s", provider.LakePrefix, uuid.New().String()),
		Hashes:   make(map[string]Photo),
	}
}

// SaveLocal writes a local file representing the current state.
func (s State) SaveLocal() error {
	json, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fmt.Sprintf("./%s", StateFile), json, 0755)
}

// SaveRemote will sync the current local state to the remote with a new UUID.
func (s State) SaveRemote(ctx context.Context, client provider.Client) error {
	s.ID = uuid.New().String()
	json, err := json.Marshal(s)
	if err != nil {
		return err
	}
	r := bytes.NewReader(json)
	_, err = client.UploadFile(ctx, StateFile, r)
	return err
}

// LocalExists determines whether or not the local version of the state file exists.
func LocalExists() (bool, error) {
	if _, err := os.Stat(fmt.Sprintf("./%s", StateFile)); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// FetchLocal will obtain the state from the flatfile.
func FetchLocal() (State, error) {
	file, err := os.OpenFile(fmt.Sprintf("./%s", StateFile), os.O_RDWR, 0755)
	if err != nil {
		return State{}, err
	}
	defer file.Close()
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return State{}, err
	}
	s := New()
	if err := json.Unmarshal(contents, &s); err != nil {
		return State{}, err
	}
	return s, err
}

// FetchRemote from provider and return and unmarshaled state object. Note that this won't
// hydrate the instance.
func FetchRemote(ctx context.Context, c provider.Client) (State, error) {
	b, err := c.DownloadFile(ctx, StateFile)
	if err != nil {
		return State{}, err
	}
	s := &State{}
	if err = json.Unmarshal(b, s); err != nil {
		return State{}, err
	}
	return *s, nil
}

// DestroyLocal will remove the local state file.
func DestroyLocal() error {
	return os.Remove(fmt.Sprintf("./%s", StateFile))
}
