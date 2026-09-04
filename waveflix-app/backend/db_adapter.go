

package main

import "database/sql"

type DatabaseAdapter interface {

	Exec(query string, args ...interface{}) (sql.Result, error)

	Query(query string, args ...interface{}) (*sql.Rows, error)

	QueryRow(query string, args ...interface{}) *sql.Row

	Begin() (*sql.Tx, error)

	DbType() string

	Ping() error

	Stats() *ConnectionStats
}

type ConnectionStats struct {

	OpenConnections int

	IdleConnections int

	InUse int

	MaxOpen int

	MaxIdle int
}
