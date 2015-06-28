package repo

import (
	"os"
	"testing"
	"time"

	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
)

const db = "/tmp/alfred-test.db"

func TestNew(t *testing.T) {
	conf.Options.DB = db
	r, err := New()
	if err != nil {
		t.Fatalf("%v", err)
	}
	r.Close()
	_, err = os.Stat(db)
	if err != nil {
		t.Fatalf("%v", err)
	}
	os.Remove(db)
}

func TestUser(t *testing.T) {
	conf.Options.DB = db
	r, err := New()
	if err != nil {
		t.Fatalf("%v", err)
	}
	err = r.SetUser(&domain.User{ID: "xxx", Name: "test", ExternalID: "yyy"})
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
	os.Remove(db)
}

func TestTeam(t *testing.T) {
	conf.Options.DB = db
	r, err := New()
	if err != nil {
		t.Fatalf("%v", err)
	}
	err = r.SetTeam(&domain.Team{ID: "xxx", Name: "test", ExternalID: "yyy"})
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
	os.Remove(db)
}

func TestTeamAndUser(t *testing.T) {
	conf.Options.DB = db
	r, err := New()
	if err != nil {
		t.Fatalf("%v", err)
	}
	err = r.SetTeamAndUser(&domain.Team{ID: "t1", Name: "test-team", ExternalID: "te1"},
		&domain.User{ID: "u1", Name: "test-user", ExternalID: "ue1"})
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
	os.Remove(db)
}

func TestOAuthState(t *testing.T) {
	conf.Options.DB = db
	r, err := New()
	if err != nil {
		t.Fatalf("%v", err)
	}
	err = r.SetOAuthState(&domain.OAuthState{State: "x", Timestamp: time.Now()})
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
	os.Remove(db)
}
