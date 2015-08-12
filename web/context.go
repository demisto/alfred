package web

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
)

// AppContext holds the web context for the handlers
type AppContext struct {
	r          repo.Repo
	q          queue.Queue
	replyQueue string
}

// NewContext creates a new context
func NewContext(r repo.Repo, q queue.Queue) *AppContext {
	host, err := os.Hostname()
	if err != nil {
		logrus.Fatal(err)
	}
	return &AppContext{r, q, host + "-web"}
}

type session struct {
	User   string    `json:"user"`
	UserID string    `json:"userId"`
	When   time.Time `json:"when"`
}
