package repo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/jmoiron/sqlx"
	// Load the mysql driver
	_ "github.com/go-sql-driver/mysql"
)

const schema = `
CREATE TABLE IF NOT EXISTS teams (
    id VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    email_domain VARCHAR(128),
		domain VARCHAR(128),
		plan VARCHAR(128),
		external_id VARCHAR(64) NOT NULL,
		created timestamp NOT NULL,
		CONSTRAINT teams_pk PRIMARY KEY (id)
);
CREATE TABLE IF NOT EXISTS users (
	id VARCHAR(64) NOT NULL,
	team VARCHAR(64) NOT NULL,
	name VARCHAR(128) NOT NULL,
	type int NOT NULL,
	status int NOT NULL,
	real_name VARCHAR(128),
	email VARCHAR(128),
	is_bot int(1) NOT NULL,
	is_admin int(1) NOT NULL,
	is_owner int(1) NOT NULL,
	is_primary_owner int(1) NOT NULL,
	is_restricted int(1) NOT NULL,
	is_ultra_restricted int(1) NOT NULL,
	external_id VARCHAR(64) NOT NULL,
	token VARCHAR(64) NOT NULL,
	created timestamp NOT NULL,
	CONSTRAINT users_pk PRIMARY KEY (id),
	CONSTRAINT users_team_fk FOREIGN KEY (team) REFERENCES teams (id),
	CONSTRAINT users_external_id_uk UNIQUE (external_id)
);
CREATE TABLE IF NOT EXISTS oauth_state (
	state VARCHAR(64) NOT NULL,
	ts TIMESTAMP NOT NULL,
	CONSTRAINT users_pk PRIMARY KEY (state)
);
CREATE TABLE IF NOT EXISTS configurations (
	user VARCHAR(64) NOT NULL,
	channel VARCHAR(64) NOT NULL,
	CONSTRAINT configurations_pk PRIMARY KEY (user, channel),
	CONSTRAINT configurations_user_fk FOREIGN KEY (user) REFERENCES users (id)
)`

type repoMySQL struct {
	db   *sqlx.DB
	stop chan bool
}

// NewMySQL repo is returned
// To create the relevant MySQL databases on local please do the following:
//   mysql -u root (if password is set then add -p)
//   mysql> CREATE DATABASE demisto CHARACTER SET = utf8;
//   mysql> CREATE DATABASE demistot CHARACTER SET = utf8;
//   mysql> CREATE USER demisto IDENTIFIED BY '***REMOVED***';
//   mysql> GRANT ALL on demisto.* TO demisto;
//   mysql> GRANT ALL on demistot.* TO demisto;
//   mysql> drop user ''@'localhost';
// The last command drops the anonymous user
func NewMySQL() (Repo, error) {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	logrus.Infof("Using MySQL at %s with user %s\n", conf.Options.DB.ConnectString, conf.Options.DB.Username)
	db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@%s", conf.Options.DB.Username, conf.Options.DB.Password, conf.Options.DB.ConnectString))
	if err != nil {
		return nil, err
	}
	creates := strings.Split(schema, ";")
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	for _, create := range creates {
		_, err = tx.Exec(create)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	r := &repoMySQL{
		db:   db,
		stop: make(chan bool),
	}
	go r.cleanOAuthState()
	return r, nil
}

func (r *repoMySQL) Close() error {
	r.stop <- true
	return r.db.Close()
}

func (r *repoMySQL) get(tableName, field, id string, data interface{}) error {
	err := r.db.Get(data, "SELECT * FROM "+tableName+" WHERE "+field+" = ?", id)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

func (r *repoMySQL) del(tableName, id string) error {
	_, err := r.db.Exec("DELETE FROM "+tableName+" WHERE id = ?", id)
	return err
}

func (r *repoMySQL) User(id string) (*domain.User, error) {
	user := &domain.User{}
	err := r.get("users", "id", id, user)
	return user, err
}

func (r *repoMySQL) UserByExternalID(id string) (*domain.User, error) {
	user := &domain.User{}
	err := r.get("users", "external_id", id, user)
	return user, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (r *repoMySQL) SetUser(user *domain.User) error {
	_, err := r.db.Exec(`INSERT INTO users
(id, team, name, type, status, real_name, email, is_bot, is_admin, is_owner, is_primary_owner, is_restricted, is_ultra_restricted, external_id, token, created)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
team = ?,
name = ?,
type = ?,
status = ?,
real_name = ?,
email = ?,
is_bot = ?,
is_admin = ?,
is_owner = ?,
is_primary_owner = ?,
is_restricted = ?,
is_ultra_restricted = ?,
external_id = ?,
token = ?,
created = ?`, user.ID, user.Team, user.Name, user.Type, user.Status, user.RealName, user.Email,
		boolToInt(user.IsBot), boolToInt(user.IsAdmin), boolToInt(user.IsOwner), boolToInt(user.IsPrimaryOwner),
		boolToInt(user.IsRestricted), boolToInt(user.IsUltraRestricted), user.ExternalID, user.Token, user.Created,
		user.Team, user.Name, user.Type, user.Status, user.RealName, user.Email, boolToInt(user.IsBot),
		boolToInt(user.IsAdmin), boolToInt(user.IsOwner), boolToInt(user.IsPrimaryOwner), boolToInt(user.IsRestricted),
		boolToInt(user.IsUltraRestricted), user.ExternalID, user.Token, user.Created)
	return err
}

func (r *repoMySQL) Team(id string) (*domain.Team, error) {
	team := &domain.Team{}
	err := r.get("teams", "id", id, team)
	return team, err
}

func (r *repoMySQL) TeamByExternalID(id string) (*domain.Team, error) {
	team := &domain.Team{}
	err := r.get("teams", "external_id", id, team)
	return team, err
}

func (r *repoMySQL) SetTeam(team *domain.Team) error {
	_, err := r.db.Exec(`INSERT INTO teams (
id, name, email_domain, domain, plan, external_id, created)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
name = ?,
email_domain = ?,
domain = ?,
plan = ?,
external_id = ?,
created = ?`,
		team.ID, team.Name, team.EmailDomain, team.Domain, team.Plan, team.ExternalID, team.Created,
		team.Name, team.EmailDomain, team.Domain, team.Plan, team.ExternalID, team.Created)
	return err
}

func (r *repoMySQL) Teams() ([]domain.Team, error) {
	var teams []domain.Team
	err := r.db.Select(&teams, "SELECT * FROM teams")
	return teams, err
}

func (r *repoMySQL) TeamMembers(team string) ([]domain.User, error) {
	var users []domain.User
	err := r.db.Select(&users, "SELECT * FROM users WHERE team = ?", team)
	return users, err
}

func (r *repoMySQL) SetTeamAndUser(team *domain.Team, user *domain.User) error {
	// TODO - too lazy right now to do transaction but this must be in transaction
	err := r.SetTeam(team)
	if err != nil {
		return err
	}
	return r.SetUser(user)
}

func (r *repoMySQL) OAuthState(id string) (*domain.OAuthState, error) {
	state := &domain.OAuthState{}
	err := r.get("oauth_state", "state", id, state)
	return state, err
}

func (r *repoMySQL) SetOAuthState(state *domain.OAuthState) error {
	_, err := r.db.Exec(`INSERT INTO oauth_state (state, ts)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE ts = ?`, state.State, state.Timestamp, state.Timestamp)
	return err
}

func (r *repoMySQL) DelOAuthState(state string) error {
	_, err := r.db.Exec("DELETE FROM oauth_state WHERE state = ?", state)
	return err
}

// cleanOAuthState deletes old states
func (r *repoMySQL) cleanOAuthState() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-r.stop:
			break
		case <-ticker.C:
			res, err := r.db.Exec("DELETE FROM oauth_state WHERE ts < ?", time.Now().Add(-5*time.Minute))
			if err != nil {
				logrus.WithField("error", err).Warnln("Unable to delete OAuth state")
				break
			} else {
				rows, err := res.RowsAffected()
				if err == nil {
					logrus.Debugf("Cleaned %v oauth states\n", rows)
				}
			}
		}
	}
}

func (r *repoMySQL) ChannelsAndGroups(user string) (*domain.Configuration, error) {
	res := &domain.Configuration{}
	var all []string
	err := r.db.Select(&all, "SELECT channel FROM configurations WHERE user = ?", user)
	for _, s := range all {
		switch s[0] {
		case 'C':
			res.Channels = append(res.Channels, s)
		case 'G':
			res.Groups = append(res.Groups, s)
		case 'D':
			res.IM = true
		case 'R':
			res.Regexp = s[1:]
		}
	}
	return res, err
}

func (r *repoMySQL) SetChannelsAndGroups(user string, configuration *domain.Configuration) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var all []string
	copy(all, configuration.Channels)
	all = append(all, configuration.Groups...)
	stmt, err := tx.Prepare("INSERT INTO configurations (user, channel) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, s := range all {
		_, err = stmt.Exec(user, s)
		if err != nil {
			return err
		}
	}
	if configuration.IM {
		_, err = stmt.Exec(user, "D")
		if err != nil {
			return err
		}
	}
	if configuration.Regexp != "" {
		_, err = stmt.Exec(user, "R"+configuration.Regexp)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *repoMySQL) TeamSubscriptions(team string) (map[string]*domain.Configuration, error) {
	subscriptions := make(map[string]*domain.Configuration)
	rows, err := r.db.Query("SELECT user, channel FROM configurations WHERE user IN (SELECT id FROM users WHERE team = ?)", team)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var user, channel string
		if err = rows.Scan(&user, &channel); err != nil {
			return nil, err
		}
		if len(channel) == 0 {
			continue
		}
		if _, ok := subscriptions[user]; !ok {
			subscriptions[user] = &domain.Configuration{}
		}
		switch channel[0] {
		case 'C':
			subscriptions[user].Channels = append(subscriptions[user].Channels, channel)
		case 'G':
			subscriptions[user].Groups = append(subscriptions[user].Groups, channel)
		case 'D':
			subscriptions[user].IM = true
		case 'R':
			subscriptions[user].Regexp = channel[1:]
		}
	}
	return subscriptions, err
}
