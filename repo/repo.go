package repo

import (
	"errors"

	"github.com/demisto/alfred/domain"
)

var (
	// ErrNotFound is a not found error if Get does not retrieve a value
	ErrNotFound = errors.New("not_found")
)

// Repo provides access to a persistent storage
type Repo interface {
	User(id string) (*domain.User, error)
	UserByExternalID(id string) (*domain.User, error)
	SetUser(user *domain.User) error
	Team(id string) (*domain.Team, error)
	TeamByExternalID(id string) (*domain.Team, error)
	Teams() ([]domain.Team, error)
	SetTeam(team *domain.Team) error
	SetTeamAndUser(team *domain.Team, user *domain.User) error
	TeamMembers(team string) ([]domain.User, error)
	OAuthState(state string) (*domain.OAuthState, error)
	SetOAuthState(state *domain.OAuthState) error
	DelOAuthState(state string) error
	ChannelsAndGroups(user string) (*domain.Configuration, error)
	SetChannelsAndGroups(user string, configuration *domain.Configuration) error
	TeamSubscriptions(team string) (map[string]*domain.Configuration, error)
	// OpenUsers retrieves all users who are currently not associated with another ACTIVE bot
	OpenUsers() ([]domain.UserBot, error)
	// LockUser associates a user to us and locks it from other bots
	LockUser(user *domain.UserBot) (bool, error)
	// BotHeartbeat updates the bot keep-alive timestamp
	BotHeartbeat() error
	UpdateStatistics(stats *domain.Statistics) error
	Statistics(team string) (*domain.Statistics, error)
	GlobalStatistics() (*domain.Statistics, error)
	Close() error
}
