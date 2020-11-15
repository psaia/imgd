package state

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/psaia/imgd/internal/provider"
	"golang.org/x/net/context"
)

type State struct {
	mu         sync.Mutex
	IsLoaded   bool           `json:"_"`
	InProgress bool           `json:"inProgress"`
	Hashes     map[string]int `json:"_ph"`
	Albums     []Album        `json:"galleries"`
}

const StateFile = ".imgd.state"

func NewState() *State {
	return &State{
		Hashes: make(map[string]int),
	}
}

func (s *State) SaveLocal() error {
	json, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fmt.Sprintf("./%s", StateFile), json, 0755)
}

func (s *State) HydrateLocal(force bool) error {
	if _, err := os.Stat(fmt.Sprintf("./%s", StateFile)); os.IsNotExist(err) {
		if force {
			if err := s.SaveLocal(); err != nil {
				return err
			}
		} else {
			return err
		}
		return nil
	}
	file, err := os.OpenFile(fmt.Sprintf("./%s", StateFile), os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(contents, s); err != nil {
		return err
	}
	s.IsLoaded = true
	return nil
}

func (*State) FetchRemote(ctx context.Context, c provider.Client) ([]byte, error) {
	b, err := c.DownloadFile(ctx, StateFile)
	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	return json, nil
}

func (s *State) DownSyncRemote(ctx context.Context, c provider.Client) error {
	b, err := s.FetchRemote(ctx, c)
	if err != nil {
		return err
	}
	json, err := json.Marshal(b)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fmt.Sprintf("./%s", StateFile), json, 0755)
}
