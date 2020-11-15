package state

type Photo struct {
	Filename string `json:"filename"`
	Hash     string `json:"hash"`
}

func (s *State) SavePhotoHash(hash string) {
	s.mu.Lock()
	s.Hashes[hash] = 1
	s.mu.Unlock()
}
