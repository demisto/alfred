package main

import (
	"flag"
	"log"

	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/repo"
)

var (
	confFile = flag.String("conf", "conf.json", "Path to configuration file in JSON format")
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	err := conf.Load(*confFile, false)
	check(err)
	r, err := repo.NewMySQL()
	check(err)
	teams, err := r.Teams()
	check(err)
	for i := range teams {
		log.Printf("Working on team %s\n", teams[i].Name)
		users, err := r.TeamMembers(teams[i].ID)
		check(err)
		for j := range users {
			log.Printf("Updating user %s\n", users[j].Name)
			err = r.SetUser(&users[j])
			check(err)
		}
	}
}
