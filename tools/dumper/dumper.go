package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
)

func main() {
	file := flag.String("file", "alfred.db", "the database file to dump")
	flag.Parse()
	db, err := bolt.Open(*file, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		panic(err)
	}
	err = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			fmt.Printf("%s:\n", string(name))
			return b.ForEach(func(k []byte, v []byte) error {
				fmt.Printf("%s:\n", string(v))
				return nil
			})
		})
	})
}
