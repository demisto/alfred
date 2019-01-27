package web

import (
	"time"

	"github.com/demisto/alfred/bot"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
)

// AppContext holds the web context for the handlers
type AppContext struct {
	r *repo.MySQL
	q queue.Queue
	b *bot.Bot
}

// NewContext creates a new context
func NewContext(r *repo.MySQL, q queue.Queue, b *bot.Bot) *AppContext {
	return &AppContext{r: r, q: q, b: b}
}

type session struct {
	User   string    `json:"user"`
	UserID string    `json:"userId"`
	When   time.Time `json:"when"`
}
