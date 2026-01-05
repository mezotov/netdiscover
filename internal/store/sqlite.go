package store

import "database/sql"

type Storage struct {
	db *sql.DB
}
