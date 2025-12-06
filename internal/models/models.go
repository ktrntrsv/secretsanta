package models

// Participant describes a player in the Secret Santa game.
type Participant struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
}

// GameData groups all persisted game information.
type GameData struct {
	Participants map[int64]Participant `json:"participants"`
	Wishlists    map[int64]string      `json:"wishlists"`
	Assignments  map[int64]int64       `json:"assignments"`
	Started      bool                  `json:"started"`
}
