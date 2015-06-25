package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
)

var (
	// ErrNotFound is a not found error if Get does not retrieve a value
	ErrNotFound = errors.New("not_found")
)

// Repo provides access to a persistent storage
type Repo interface {
	User(id string) (*domain.User, error)
	SetUser(user *domain.User) error
	Team(id string) (*domain.Team, error)
	SetTeam(team *domain.Team) error
	SetTeamAndUser(team *domain.Team, user *domain.User) error
	GetTeamMembers(team string) ([]domain.User, error)
	OAuthState(state string) (*domain.OAuthState, error)
	SetOAuthState(state *domain.OAuthState) error
	DelOAuthState(state string) error
	ChannelsAndGroups(user string) (*domain.Configuration, error)
	SetChannelsAndGroups(user string, configuration *domain.Configuration) error
	Close()
}

type repo struct {
	db *bolt.DB
}

// New repo is returned
func New() (Repo, error) {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	logrus.Infof("Using database file %s\n", conf.Options.DB)
	db, err := bolt.Open(conf.Options.DB, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("teams"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("oauth"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("channels"))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	r := &repo{db}
	go r.cleanOAuthState()
	return r, nil
}

func (r *repo) Close() {
	r.db.Close()
}

func (r *repo) get(bucket, key string, data interface{}) error {
	return r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		v := b.Get([]byte(key))
		if v == nil {
			return ErrNotFound
		}
		err := json.Unmarshal(v, data)
		return err
	})
}

func (r *repo) set(bucket, key string, data interface{}) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		v, err := json.Marshal(data)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), v)
	})
}

func (r *repo) del(bucket, key string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Delete([]byte(key))
	})
}

func (r *repo) User(id string) (*domain.User, error) {
	user := &domain.User{}
	err := r.get("users", id, user)
	return user, err
}

func (r *repo) SetUser(user *domain.User) error {
	return r.set("users", user.ID, user)
}

func (r *repo) Team(id string) (*domain.Team, error) {
	team := &domain.Team{}
	err := r.get("teams", id, team)
	return team, err
}

func (r *repo) SetTeam(team *domain.Team) error {
	return r.set("teams", team.ID, team)
}

func (r *repo) GetTeamMembers(team string) ([]domain.User, error) {
	var users []domain.User
	err := r.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("teams")).Cursor()
		b := tx.Bucket([]byte("users"))
		prefix := []byte(team + "#")
		for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			id := k[len(prefix):]
			u := b.Get(id)
			var user domain.User
			err := json.Unmarshal(u, &user)
			if err != nil {
				return err
			}
			users = append(users, user)
		}
		return nil
	})
	return users, err
}

func (r *repo) SetTeamAndUser(team *domain.Team, user *domain.User) error {
	err := r.db.Batch(func(tx *bolt.Tx) error {
		ub := tx.Bucket([]byte("users"))
		tb := tx.Bucket([]byte("teams"))
		tv, err := json.Marshal(team)
		if err != nil {
			return err
		}
		uv, err := json.Marshal(user)
		if err != nil {
			return err
		}
		err = tb.Put([]byte(team.ID), tv)
		if err != nil {
			return err
		}
		// Index the user in the team
		err = tb.Put([]byte(team.ID+"#"+user.ID), []byte{})
		_, err = tx.CreateBucketIfNotExists([]byte("oauth"))
		if err != nil {
			return err
		}
		err = ub.Put([]byte(user.ID), uv)
		return err
	})
	return err
}

func (r *repo) OAuthState(id string) (*domain.OAuthState, error) {
	state := &domain.OAuthState{}
	err := r.get("oauth", id, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (r *repo) SetOAuthState(state *domain.OAuthState) error {
	return r.set("oauth", state.State, state)
}

func (r *repo) DelOAuthState(state string) error {
	return r.del("oauth", state)
}

// cleanOAuthState deletes old states
func (r *repo) cleanOAuthState() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := r.db.Batch(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("oauth"))
				return b.ForEach(func(k []byte, v []byte) error {
					var state domain.OAuthState
					err := json.Unmarshal(v, &state)
					if err != nil {
						return err
					}
					if time.Since(state.Timestamp) > 5*time.Minute {
						err = b.Delete(k)
						if err != nil {
							return err
						}
					}
					return nil
				})
			})
			if err != nil {
				logrus.WithField("error", err).Warnln("Unable to delete OAuth state")
				break
			}
		}
	}
}

func (r *repo) ChannelsAndGroups(user string) (*domain.Configuration, error) {
	res := &domain.Configuration{}
	err := r.get("channels", user, res)
	if err == ErrNotFound {
		err = nil
	}
	return res, err
}

func (r *repo) SetChannelsAndGroups(user string, configuration *domain.Configuration) error {
	return r.set("channels", user, configuration)
}
