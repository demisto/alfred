package web

import (
	"time"

	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
)

// AppContext holds the web context for the handlers
type AppContext struct {
	r repo.Repo
	q queue.Queue
}

// NewContext creates a new context
func NewContext(r repo.Repo, q queue.Queue) *AppContext {
	return &AppContext{r, q}
}

type session struct {
	User   string    `json:"user"`
	UserID string    `json:"userId"`
	When   time.Time `json:"when"`
}
