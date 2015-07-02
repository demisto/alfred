package repo

import (
	"encoding/json"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/util"
)

type repo struct {
	db *bolt.DB
}

// New repo is returned
func New() (Repo, error) {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	logrus.Infof("Using database file %s\n", conf.Options.DB.ConnectString)
	db, err := bolt.Open(conf.Options.DB.ConnectString, 0600, &bolt.Options{Timeout: 1 * time.Second})
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
		_, err = tx.CreateBucketIfNotExists([]byte("teamusers"))
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

func (r *repo) UserByExternalID(id string) (*domain.User, error) {
	var user *domain.User
	err := r.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("users")).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var u domain.User
			err := json.Unmarshal(v, &u)
			if err != nil {
				return err
			}
			if u.ExternalID == id {
				user = &u
				break
			}
		}
		return nil
	})
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

func (r *repo) TeamByExternalID(id string) (*domain.Team, error) {
	var team *domain.Team
	err := r.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("teams")).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var t domain.Team
			err := json.Unmarshal(v, &t)
			if err != nil {
				return err
			}
			if t.ExternalID == id {
				team = &t
				break
			}
		}
		return nil
	})
	return team, err
}

func (r *repo) SetTeam(team *domain.Team) error {
	return r.set("teams", team.ID, team)
}

func (r *repo) Teams() ([]domain.Team, error) {
	var teams []domain.Team
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("teams"))
		return b.ForEach(func(k []byte, v []byte) error {
			var team domain.Team
			err := json.Unmarshal(v, &team)
			if err != nil {
				return err
			}
			teams = append(teams, team)
			return nil
		})
	})
	return teams, err
}

func (r *repo) TeamMembers(team string) ([]domain.User, error) {
	var users []domain.User
	err := r.db.View(func(tx *bolt.Tx) error {
		tb := tx.Bucket([]byte("teamusers"))
		ub := tx.Bucket([]byte("users"))
		var ids []string
		members := tb.Get([]byte(team))
		if members == nil {
			return nil
		}
		err := json.Unmarshal(members, &ids)
		if err != nil {
			return err
		}
		for _, id := range ids {
			u := ub.Get([]byte(id))
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
		tub := tx.Bucket([]byte("teamusers"))
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
		err = ub.Put([]byte(user.ID), uv)
		if err != nil {
			return err
		}
		var ids []string
		members := tub.Get([]byte(team.ID))
		if members != nil {
			err = json.Unmarshal(members, &ids)
			if err != nil {
				return err
			}
		}
		if !util.In(ids, user.ID) {
			ids = append(ids, user.ID)
			members, err = json.Marshal(&ids)
			if err != nil {
				return err
			}
			return tub.Put([]byte(team.ID), members)
		}
		return nil
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
				c := tx.Bucket([]byte("oauth")).Cursor()
				for k, v := c.First(); k != nil; k, v = c.Next() {
					var state domain.OAuthState
					err := json.Unmarshal(v, &state)
					if err != nil {
						return err
					}
					if time.Since(state.Timestamp) > 5*time.Minute {
						err = c.Delete()
						if err != nil {
							return err
						}
					}
				}
				return nil
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
