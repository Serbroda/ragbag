package db

import (
	"database/sql"
	_ "github.com/glebarez/sqlite"
	"sync"
)

var once sync.Once

var (
	DB *sql.DB
)

func OpenConnection(driver, source string) *sql.DB {
	once.Do(func() {
		db, err := connectDatabase(driver, source)
		if err != nil {
			panic("Failed to open database: " + err.Error())
		}
		DB = db
	})
	return DB
}

func connectDatabase(driver, source string) (*sql.DB, error) {
	db, err := sql.Open(driver, source)
	if err != nil {
		return nil, err
	}
	return db, nil
}
