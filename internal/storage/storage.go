package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"secret-santa/internal/models"
)

// Store persists game data into JSON files.
type Store struct {
	dir string
	mu  sync.Mutex
}

const (
	participantsFile = "participants.json"
	wishlistsFile    = "wishlists.json"
	assignmentsFile  = "assignments.json"
	stateFile        = "state.json"
)

// NewStore prepares a Store for the provided directory.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// LoadAll reads all persisted data. Missing files are treated as empty data.
func (s *Store) LoadAll() (*models.GameData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := &models.GameData{
		Participants: make(map[int64]models.Participant),
		Wishlists:    make(map[int64]string),
		Assignments:  make(map[int64]int64),
		Started:      false,
	}

	readers := []struct {
		filename string
		target   any
	}{
		{participantsFile, &data.Participants},
		{wishlistsFile, &data.Wishlists},
		{assignmentsFile, &data.Assignments},
	}

	for _, r := range readers {
		path := filepath.Join(s.dir, r.filename)
		if err := readJSON(path, r.target); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
	}

	statePath := filepath.Join(s.dir, stateFile)
	var state struct {
		Started bool `json:"started"`
	}
	if err := readJSON(statePath, &state); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	data.Started = state.Started

	return data, nil
}

// SaveParticipants writes participants to disk.
func (s *Store) SaveParticipants(participants map[int64]models.Participant) error {
	return s.writeFile(participantsFile, participants)
}

// SaveWishlists writes wishlists to disk.
func (s *Store) SaveWishlists(wishlists map[int64]string) error {
	return s.writeFile(wishlistsFile, wishlists)
}

// SaveAssignments writes assignments to disk.
func (s *Store) SaveAssignments(assignments map[int64]int64) error {
	return s.writeFile(assignmentsFile, assignments)
}

// SaveState persists overall state (currently only Started flag).
func (s *Store) SaveState(started bool) error {
	state := struct {
		Started bool `json:"started"`
	}{
		Started: started,
	}
	return s.writeFile(stateFile, state)
}

func (s *Store) writeFile(name string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, name)
	tmp, err := os.CreateTemp(s.dir, "tmp-*.json")
	if err != nil {
		return err
	}

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
