package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"secret-santa/internal/models"
	"secret-santa/internal/storage"
	"sync"
)

var (
	ErrAlreadyJoined  = errors.New("already joined")
	ErrGameStarted    = errors.New("game already started")
	ErrNotParticipant = errors.New("not participant")
	ErrTooFewPlayers  = errors.New("not enough players")
	ErrGameNotStarted = errors.New("game not started")
	ErrAlreadyStarted = errors.New("already started")
)

// Game keeps the in-memory state and coordinates persistence.
type Game struct {
	mu           sync.RWMutex
	store        *storage.Store
	participants map[int64]models.Participant
	wishlists    map[int64]string
	assignments  map[int64]int64
	santas       map[int64]int64 // giftee -> santa
	started      bool
}

// NewGame loads data from store and prepares the game.
func NewGame(store *storage.Store) (*Game, error) {
	data, err := store.LoadAll()
	if err != nil {
		return nil, err
	}

	g := &Game{
		store:        store,
		participants: data.Participants,
		wishlists:    data.Wishlists,
		assignments:  data.Assignments,
		started:      data.Started || len(data.Assignments) > 0,
	}
	g.buildSantaMap()
	return g, nil
}

// Join adds a participant before the game starts.
func (g *Game) Join(p models.Participant) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.started {
		return ErrGameStarted
	}
	if _, ok := g.participants[p.UserID]; ok {
		return ErrAlreadyJoined
	}

	g.participants[p.UserID] = p
	return g.store.SaveParticipants(g.participants)
}

// Leave removes a participant if the game hasn't started.
func (g *Game) Leave(userID int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.started {
		return ErrGameStarted
	}
	if _, ok := g.participants[userID]; !ok {
		return ErrNotParticipant
	}

	delete(g.participants, userID)
	return g.store.SaveParticipants(g.participants)
}

// UpdateWishlist sets or updates a wishlist. Returns santaID if someone should be notified.
func (g *Game) UpdateWishlist(userID int64, text string) (santaID int64, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.wishlists[userID] = text
	if err := g.store.SaveWishlists(g.wishlists); err != nil {
		return 0, err
	}

	if g.started && g.santas != nil {
		if santa, ok := g.santas[userID]; ok {
			return santa, nil
		}
	}
	return 0, nil
}

// Start creates random assignments and locks the game.
func (g *Game) Start() (map[int64]int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.started {
		return nil, ErrAlreadyStarted
	}

	if len(g.participants) < 2 {
		return nil, ErrTooFewPlayers
	}

	ids := make([]int64, 0, len(g.participants))
	for id := range g.participants {
		ids = append(ids, id)
	}

	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})

	assignments := make(map[int64]int64, len(ids))
	for i := range ids {
		giver := ids[i]
		receiver := ids[(i+1)%len(ids)]
		assignments[giver] = receiver
	}

	g.assignments = assignments
	g.started = true
	g.buildSantaMap()

	if err := g.store.SaveAssignments(g.assignments); err != nil {
		return nil, err
	}
	if err := g.store.SaveState(g.started); err != nil {
		return nil, err
	}

	return assignments, nil
}

// GetGiftee returns the participant this user should gift to.
func (g *Game) GetGiftee(userID int64) (models.Participant, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.started {
		return models.Participant{}, ErrGameNotStarted
	}

	gifteeID, ok := g.assignments[userID]
	if !ok {
		return models.Participant{}, fmt.Errorf("assignment not found")
	}

	p, ok := g.participants[gifteeID]
	if !ok {
		return models.Participant{}, fmt.Errorf("participant missing")
	}
	return p, nil
}

// GetSanta returns the santa for a giftee.
func (g *Game) GetSanta(gifteeID int64) (int64, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.started {
		return 0, ErrGameNotStarted
	}

	if santa, ok := g.santas[gifteeID]; ok {
		return santa, nil
	}
	return 0, fmt.Errorf("santa not found")
}

// GetWishlist returns a wishlist text.
func (g *Game) GetWishlist(userID int64) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	text, ok := g.wishlists[userID]
	return text, ok
}

// IsStarted tells if the game is locked.
func (g *Game) IsStarted() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.started
}

// IsParticipant checks membership.
func (g *Game) IsParticipant(userID int64) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.participants[userID]
	return ok
}

// Participants returns a snapshot of current participants.
func (g *Game) Participants() map[int64]models.Participant {
	g.mu.RLock()
	defer g.mu.RUnlock()

	copyMap := make(map[int64]models.Participant, len(g.participants))
	for id, p := range g.participants {
		copyMap[id] = p
	}
	return copyMap
}

func (g *Game) buildSantaMap() {
	santas := make(map[int64]int64, len(g.assignments))
	for santa, giftee := range g.assignments {
		santas[giftee] = santa
	}
	g.santas = santas
}
