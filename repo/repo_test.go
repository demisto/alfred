// +build integration

package repo

import (
	"testing"
	"time"

	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/util"
)

func getTestDB(t *testing.T) *MySQL {
	conf.Load("", true)
	conf.Options.DB.ConnectString, conf.Options.DB.Username, conf.Options.DB.Password = "tcp/demistot?parseTime=true", "demisto", "demisto1999"
	db, err := NewMySQL()
	if err != nil {
		t.Fatalf("%v", err)
	}
	db.db.Exec("DELETE FROM queue")
	db.db.Exec("DELETE FROM convicted")
	db.db.Exec("DELETE FROM slack_invites")
	db.db.Exec("DELETE FROM team_statistics")
	db.db.Exec("DELETE FROM bot_for_team")
	db.db.Exec("DELETE FROM bots")
	db.db.Exec("DELETE FROM configuration")
	db.db.Exec("DELETE FROM oauth_state")
	db.db.Exec("DELETE FROM users")
	db.db.Exec("DELETE FROM teams")
	return db
}

func TestNewMySQL(t *testing.T) {
	r := getTestDB(t)
	r.Close()
}

func TestUserMySQL(t *testing.T) {
	r := getTestDB(t)
	err := r.SetTeam(&domain.Team{ID: "zzz", Name: "test"})
	if err != nil {
		t.Errorf("Unable to create team - %v", err)
	}
	err = r.SetUser(&domain.User{ID: "xxx", Team: "zzz", Name: "test", ExternalID: "yyy"})
	if err != nil {
		t.Errorf("Unable to create user - %v", err)
	}
	u, err := r.User("xxx")
	if err != nil {
		t.Errorf("Unable to load user - %v", err)
	}
	u, err = r.UserByExternalID("yyy")
	if err != nil {
		t.Errorf("Unable to load user by external ID - %v", err)
	}
	if u.Name != "test" {
		t.Error("User name is not retrieved")
	}
	r.Close()
}

func TestTeamMySQL(t *testing.T) {
	r := getTestDB(t)
	err := r.SetTeam(&domain.Team{ID: "xxx", Name: "test", ExternalID: "yyy"})
	if err != nil {
		t.Errorf("Unable to create team - %v", err)
	}
	team, err := r.Team("xxx")
	if err != nil {
		t.Errorf("Unable to load team - %v", err)
	}
	team, err = r.TeamByExternalID("yyy")
	if err != nil {
		t.Errorf("Unable to load team by external ID - %v", err)
	}
	if team.Name != "test" {
		t.Error("Team name is not retrieved")
	}
	teams, err := r.Teams()
	if err != nil {
		t.Errorf("Unable to load teams - %v", err)
	}
	if len(teams) != 1 {
		t.Errorf("expecting only 1 team but got %d", len(teams))
	}
	r.Close()
}

func TestTeamAndUserMySQL(t *testing.T) {
	r := getTestDB(t)
	err := r.SetTeamAndUser(&domain.Team{ID: "t1", Name: "test-team", ExternalID: "te1"},
		&domain.User{ID: "u1", Team: "t1", Name: "test-user", ExternalID: "ue1"})
	if err != nil {
		t.Errorf("Unable to create team and user - %v", err)
	}
	team, err := r.Team("t1")
	if err != nil {
		t.Errorf("Unable to load team - %v", err)
	}
	u, err := r.User("u1")
	if err != nil {
		t.Errorf("Unable to load user - %v", err)
	}
	users, err := r.TeamMembers(team.ID)
	if err != nil {
		t.Errorf("Unable to load team members - %v", err)
	}
	if len(users) != 1 || users[0].ID != u.ID {
		t.Error("Did not load the correct user")
	}
	r.Close()
}

func TestOAuthStateMySQL(t *testing.T) {
	r := getTestDB(t)
	err := r.SetOAuthState(&domain.OAuthState{State: "x", Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Unable to save state - %v", err)
	}
	s, err := r.OAuthState("x")
	if err != nil {
		t.Errorf("Unable to load state - %v", err)
	}
	err = r.DelOAuthState(s.State)
	if err != nil {
		t.Errorf("Unable to load state - %v", err)
	}
	r.Close()
}

func TestQueueMessages(t *testing.T) {
	r := getTestDB(t)
	messages, err := r.QueueMessages(false, "work")
	if err != nil {
		t.Fatalf("Unable to load messages - %v", err)
	}
	if len(messages) > 0 {
		t.Errorf("Got messages but expecting none")
	}
	err = r.SetTeam(&domain.Team{ID: "kuku", Name: "kuku", ExternalID: "xxx", BotToken: "xxx", BotUserID: "xxx"})
	if err != nil {
		t.Errorf("Unable to create team - %v", err)
	}
	err = r.PostMessage(&domain.DBQueueMessage{Team: "kuku", Message: "ABC", MessageType: "work"})
	if err != nil {
		t.Errorf("Unable to post message - %v", err)
	}
	messages, err = r.QueueMessages(false, "work")
	if err != nil {
		t.Fatalf("Unable to load messages - %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Expecting 1 message but got %d", len(messages))
	} else {
		if messages[0].MessageType != "work" || messages[0].Team != "kuku" || messages[0].Message != "ABC" {
			t.Errorf("Got wrong data for message %s", util.ToJSONString(messages[0]))
		}
	}
	messages, err = r.QueueMessages(false, "work")
	if err != nil {
		t.Fatalf("Unable to load messages - %v", err)
	}
	if len(messages) > 0 {
		t.Errorf("Got messages but expecting none after delete")
	}
}
