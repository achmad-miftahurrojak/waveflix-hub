package main

import (
	_ "github.com/jackc/pgx/v5/stdlib"
)

// This file ensures pgx dependencies are retained in go.mod
// The pgx/v5/stdlib package will be used for PostgreSQL database/sql driver
