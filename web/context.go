package web

import (
	"time"

	"github.com/demisto/alfred/bot"
	"github.com/demisto/alfred/repo"
)

// AppContext holds the web context for the handlers
type AppContext struct {
	r repo.Repo
	b *bot.Bot
}

// NewContext creates a new context
func NewContext(r repo.Repo, b *bot.Bot) *AppContext {
	return &AppContext{r, b}
}

type session struct {
	User   string    `json:"user"`
	UserID string    `json:"userId"`
	When   time.Time `json:"when"`
}
