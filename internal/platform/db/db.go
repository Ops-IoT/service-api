package db

import (
	"database/sql"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
)

//DB : struct type holding sql.DB type
type DB struct {
	Connection *sql.DB
}

//New : returns new DB struct
func New(host, name, user, pass string, maxIdle, maxActive int) (DB, error) {

	var db DB
	conn, err := sql.Open("mysql", url)
	if err != nil {
		return db, err
	}

	conn.SetMaxIdleConns(maxIdle)
	conn.SetMaxOpenConns(maxActive)

	err = db.Ping()
	if err != nil {
		return db, err
	}

	db.Connection = conn
	return db, nil
}

//Close : closes database connection
func (db *DB) Close() error {
	return db.Connection.Close()
}
