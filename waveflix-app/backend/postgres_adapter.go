

package main

import (
"database/sql"
)

type postgresAdapter struct {
db *sql.DB
}

func newPostgresAdapter(db *sql.DB) DatabaseAdapter {
return &postgresAdapter{db: db}
}

func (p *postgresAdapter) Exec(query string, args ...interface{}) (sql.Result, error) {
return p.db.Exec(query, args...)
}

func (p *postgresAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
return p.db.Query(query, args...)
}

func (p *postgresAdapter) QueryRow(query string, args ...interface{}) *sql.Row {
return p.db.QueryRow(query, args...)
}

func (p *postgresAdapter) Begin() (*sql.Tx, error) {
return p.db.Begin()
}

func (p *postgresAdapter) DbType() string {
return "postgres"
}

func (p *postgresAdapter) Ping() error {
return p.db.Ping()
}

func (p *postgresAdapter) Stats() *ConnectionStats {
dbStats := p.db.Stats()

return &ConnectionStats{
OpenConnections: dbStats.OpenConnections,
IdleConnections: dbStats.Idle,
InUse:           dbStats.InUse,
MaxOpen:         dbStats.MaxOpenConnections,
MaxIdle:         dbStats.MaxOpenConnections, 
}
}
