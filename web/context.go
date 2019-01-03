package web

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/bot"
	"github.com/demisto/alfred/queue"
	"github.com/demisto/alfred/repo"
	"github.com/demisto/alfred/util"
)

// AppContext holds the web context for the handlers
type AppContext struct {
	r          *repo.MySQL
	q          queue.Queue
	replyQueue string
	b          *bot.Bot
}

// NewContext creates a new context
func NewContext(r *repo.MySQL, q queue.Queue, b *bot.Bot) *AppContext {
	host, err := queue.ReplyQueueName()
	if err != nil {
		logrus.Fatal(err)
	}
	return &AppContext{r: r, q: q, replyQueue: host + util.WebReplySuffix, b: b}
}

type session struct {
	User   string    `json:"user"`
	UserID string    `json:"userId"`
	When   time.Time `json:"when"`
}
