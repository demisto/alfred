package repo

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
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
);
CREATE TABLE IF NOT EXISTS bots (
	bot VARCHAR(64) NOT NULL,
	ts TIMESTAMP NOT NULL,
	CONSTRAINT bots_pk PRIMARY KEY (bot)
);
CREATE TABLE IF NOT EXISTS bot_for_user (
	user VARCHAR(64) NOT NULL,
	bot VARCHAR(64) NOT NULL,
	ts TIMESTAMP NOT NULL,
	CONSTRAINT bot_for_user_pk PRIMARY KEY (user),
	CONSTRAINT bot_for_user_u_fk FOREIGN KEY (user) REFERENCES users(id),
	CONSTRAINT bot_for_user_b_fk FOREIGN KEY (bot) REFERENCES bots(bot)
);
CREATE TABLE IF NOT EXISTS team_statistics (
	team VARCHAR(64) NOT NULL,
	ts TIMESTAMP NOT NULL,
	messages BIGINT NOT NULL,
	files_clean BIGINT NOT NULL,
	files_dirty BIGINT NOT NULL,
	files_unknown BIGINT NOT NULL,
	urls_clean BIGINT NOT NULL,
	urls_dirty BIGINT NOT NULL,
	urls_unknown BIGINT NOT NULL,
	hashes_clean BIGINT NOT NULL,
	hashes_dirty BIGINT NOT NULL,
	hashes_unknown BIGINT NOT NULL,
	ips_clean BIGINT NOT NULL,
	ips_dirty BIGINT NOT NULL,
	ips_unknown BIGINT NOT NULL,
	CONSTRAINT team_statistics_pk PRIMARY KEY (team),
	CONSTRAINT team_statistics_team_fk FOREIGN KEY (team) REFERENCES teams (id)
)`

type repoMySQL struct {
	db       *sqlx.DB
	hostname string
	stop     chan bool
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
	logrus.Infof("Using MySQL at %s with user %s\n", conf.Options.DB.ConnectString, conf.Options.DB.Username)
	name, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	// If we specified TLS connection, we need the certificate files
	if conf.Options.DB.ServerCA != "" {
		rootCertPool := x509.NewCertPool()
		if ok := rootCertPool.AppendCertsFromPEM([]byte(conf.Options.DB.ServerCA)); !ok {
			return nil, errors.New("Unable to add ServerCA PEM")
		}
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.X509KeyPair([]byte(conf.Options.DB.ClientCert), []byte(conf.Options.DB.ClientKey))
		if err != nil {
			return nil, err
		}
		clientCert = append(clientCert, certs)
		mysql.RegisterTLSConfig("dbot", &tls.Config{
			RootCAs:            rootCertPool,
			Certificates:       clientCert,
			InsecureSkipVerify: true,
		})
	}
	db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@%s", conf.Options.DB.Username, conf.Options.DB.Password, conf.Options.DB.ConnectString))
	if err != nil {
		return nil, err
	}
	// Have to set it to make sure no connection is left idle and being killed
	db.SetMaxIdleConns(0)
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
		db:       db,
		hostname: name,
		stop:     make(chan bool),
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
		case 'A':
			res.All = true
		}
	}
	return res, err
}

func (r *repoMySQL) SetChannelsAndGroups(user string, configuration *domain.Configuration) error {
	logrus.Debugf("Saving configuration for user %s - %+v\n", user, configuration)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var all []string
	all = append(all, configuration.Channels...)
	all = append(all, configuration.Groups...)
	// First, delete the configuration for the user
	_, err = tx.Exec("DELETE FROM configurations WHERE user = ?", user)
	if err != nil {
		return err
	}
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
	if configuration.All {
		_, err = stmt.Exec(user, "A")
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
	defer rows.Close()
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
		case 'A':
			subscriptions[user].All = true
		}
	}
	return subscriptions, err
}

func (r *repoMySQL) OpenUsers() ([]domain.UserBot, error) {
	var users []domain.UserBot
	rows, err := r.db.Query(
		"SELECT u.id as user, ub.bot, ub.ts FROM users u LEFT OUTER JOIN bot_for_user ub ON u.id = ub.user LEFT OUTER JOIN bots b ON ub.bot = b.bot WHERE ub.bot IS NULL OR b.ts + interval ? minute < now() LIMIT 1000", 3)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var user string
		var bot sql.NullString
		var ts mysql.NullTime
		if err = rows.Scan(&user, &bot, &ts); err != nil {
			return nil, err
		}
		u := domain.UserBot{User: user}
		if bot.Valid {
			u.Bot = bot.String
		}
		if ts.Valid {
			u.Timestamp = ts.Time
		}
		users = append(users, u)
	}
	return users, err
}

func (r *repoMySQL) LockUser(user *domain.UserBot) (bool, error) {
	// This line does not exist
	if user.Bot == "" {
		_, err := r.db.Exec("INSERT INTO bot_for_user (user, bot, ts) VALUES (?, ?, now())", user.User, r.hostname)
		if err != nil {
			switch err := err.(type) {
			case *mysql.MySQLError:
				// Duplicate key is expected so just return false
				if err.Number == 1062 {
					return false, nil
				}
			}
			return false, err
		}
	}
	result, err := r.db.Exec("UPDATE bot_for_user SET bot = ?, ts = now() WHERE user = ? AND bot = ? AND ts = ?", r.hostname, user.User, user.Bot, user.Timestamp)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

func (r *repoMySQL) BotHeartbeat() error {
	_, err := r.db.Exec("INSERT INTO bots (bot, ts) VALUES (?, now()) ON DUPLICATE KEY UPDATE ts = now()", r.hostname)
	return err
}

func (r *repoMySQL) updateStats(stats *domain.Statistics, oldTimestamp time.Time) error {
	var rows int64
	for count := 5; rows == 0 && count > 0; count-- {
		res, err := r.db.Exec(`UPDATE team_statistics SET
ts = now(),
messages = messages + ?,
files_clean = files_clean + ?,
files_dirty = files_dirty + ?,
files_unknown = files_unknown + ?,
urls_clean = urls_clean + ?,
urls_dirty = urls_dirty + ?,
urls_unknown = urls_unknown + ?,
hashes_clean = hashes_clean + ?,
hashes_dirty = hashes_dirty + ?,
hashes_unknown = hashes_unknown + ?,
ips_clean = ips_clean + ?,
ips_dirty = ips_dirty + ?,
ips_unknown = ips_unknown + ?
WHERE team = ? AND ts = ?`,
			stats.Messages, stats.FilesClean, stats.FilesDirty, stats.FilesUnknown, stats.URLsClean, stats.URLsDirty, stats.URLsUnknown,
			stats.HashesClean, stats.HashesDirty, stats.HashesUnknown, stats.IPsClean, stats.IPsDirty, stats.IPsUnknown, stats.Team, oldTimestamp)
		if err != nil {
			return err
		}
		rows, err = res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			err = r.db.Get(&oldTimestamp, "SELECT ts FROM team_statistics WHERE team = ?", stats.Team)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *repoMySQL) UpdateStatistics(stats *domain.Statistics) error {
	// Can be probably done via UPSERT
	// The code selects current timestamp. If there is no row for the team, we try to insert. If insert fails (because someone already inserted this team) then move to updates.
	// The updates try to update the row while making sure that the timestamp is the same as we selected. If someone changed data, we will need to re-select timestmap to prevent lost updates.
	var oldTimestamp time.Time
	err := r.db.Get(&oldTimestamp, "SELECT ts FROM team_statistics WHERE team = ?", stats.Team)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		_, err := r.db.Exec(`INSERT INTO team_statistics
(team, ts, messages, files_clean, files_dirty, files_unknown, urls_clean, urls_dirty, urls_unknown, hashes_clean, hashes_dirty, hashes_unknown, ips_clean, ips_dirty, ips_unknown)
VALUES (?, now(), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			stats.Team, stats.Messages, stats.FilesClean, stats.FilesDirty, stats.FilesUnknown, stats.URLsClean, stats.URLsDirty, stats.URLsUnknown,
			stats.HashesClean, stats.HashesDirty, stats.HashesUnknown, stats.IPsClean, stats.IPsDirty, stats.IPsUnknown)
		if err != nil {
			switch mysqlErr := err.(type) {
			case *mysql.MySQLError:
				// Duplicate key because someone already inserted stats for team
				if mysqlErr.Number == 1062 {
					// Do select again and then update
					err = r.db.Get(&oldTimestamp, "SELECT ts FROM team_statistics WHERE team = ?", stats.Team)
					if err != nil {
						return err
					}
					return r.updateStats(stats, oldTimestamp)
				}
			}
			return err
		}
		return nil
	}
	return r.updateStats(stats, oldTimestamp)
}

func (r *repoMySQL) Statistics(team string) (*domain.Statistics, error) {
	stats := &domain.Statistics{}
	err := r.db.Get(stats, "SELECT * FROM team_statistics WHERE team = ?", team)
	return stats, err
}

func (r *repoMySQL) GlobalStatistics() (*domain.Statistics, error) {
	// Notice - this will not work if there are no statistics at all in the DB
	stats := &domain.Statistics{}
	err := r.db.Get(stats, `SELECT 'Global' as team, sum(messages) as messages,
sum(files_clean) as clean_files, sum(files_dirty) as files_dirty, sum(files_unknown) as files_unknown,
sum(urls_clean) as urls_clean, sum(urls_dirty) as urls_dirty, sum(urls_unknown) as urls_unknown,
sum(hashes_clean) as hashes_clean, sum(hashes_dirty) as hashes_dirty, sum(hashes_unknown) as hashes_unknown,
sum(ips_clean) as ips_clean, sum(ips_dirty) as ips_dirty, sum(ips_unknown) as ips_unknown FROM team_statistics`)
	return stats, err
}
