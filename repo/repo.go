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
	BotName() string
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
	ChannelsAndGroups(team string) (*domain.Configuration, error)
	SetChannelsAndGroups(team string, configuration *domain.Configuration) error
	IsVerboseChannel(team, channel string) (bool, error)
	// OpenTeams retrieves all teams who are currently not associated with another ACTIVE bot
	OpenTeams(includeMine bool) ([]domain.TeamBot, error)
	// LockUser associates a user to us and locks it from other bots
	LockTeam(team *domain.TeamBot) (bool, error)
	// Unlock the team as it is being deleted
	UnlockTeam(id string) error
	// BotHeartbeat updates the bot keep-alive timestamp
	BotHeartbeat() error
	UpdateStatistics(stats *domain.Statistics) error
	Statistics(team string) (*domain.Statistics, error)
	GlobalStatistics() (*domain.Statistics, error)
	TotalMessages() (int, error)
	// StoreMaliciousContent in the DB
	StoreMaliciousContent(convicted *domain.MaliciousContent) error
	JoinSlackChannel(email string) error
	Close() error
}
