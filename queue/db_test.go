// +build integration

package queue

import (
	"testing"

	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/demisto/alfred/repo"
	"github.com/stretchr/testify/assert"
)

func getTestDB(t *testing.T) *repo.MySQL {
	conf.Load("", true)
	conf.Options.DB.ConnectString, conf.Options.DB.Username, conf.Options.DB.Password = "tcp/demistot?parseTime=true", "demisto", "demisto1999"
	db, err := repo.NewMySQL()
	if err != nil {
		t.Fatal(err)
	}
	db.db.Exec("DELETE FROM queue")
	db.db.Exec("DELETE FROM teams")
	return db
}

func TestDbQueue_PushWork(t *testing.T) {
	r := getTestDB(t)
	q, err := New(r)
	if err != nil {
		t.Fatal(err)
	}
	err = r.SetTeam(&domain.Team{ID: "kuku", Name: "kuku"})
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, q.PushWork(&domain.WorkRequest{Text: "kuku", Type: "message"}))

	defer q.Close()
}
