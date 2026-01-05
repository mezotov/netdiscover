package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/mezotov/netdiscover/internal/model"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS devices (
	    id TEXT PRIMARY KEY,
	    mac TEXT,
	    ips TEXT,
	    hostnames TEXT,
	    last_seen integer
	)
	`)
	return err
}

func (s *SQLiteStore) Get(id string) (*model.Device, bool) {
	row := s.db.QueryRow(`SELECT mac, ips, hostnames, last_seen FROM devices WHERE id = ?`, id)

	var mac, ips, hostnames string
	var lastSeen int64

	if err := row.Scan(&mac, &ips, &hostnames, &lastSeen); err != nil {
		return nil, false
	}

	d := model.NewDevice(id)
	d.LastSeen = time.Unix(lastSeen, 0)

	json.Unmarshal([]byte(ips), &d.IPs)
	json.Unmarshal([]byte(hostnames), &d.Hostnames)

	return d, true
}

func (s *SQLiteStore) Save(d *model.Device) {
	ips, _ := json.Marshal(d.IPs)
	names, _ := json.Marshal(d.Hostnames)

	s.db.Exec(`
	INSERT INTO devices (mac, ips, hostnames, last_seen)
	VALUES (?, ?, ?, ?)
	ON CONFLICT (id) DO UPDATE SET
	                    ips = excluded.ips,
	                    hostnames = excluded.hostnames,
	                    last_seen = excluded.last_seen
	`,
		d.ID,
		d.MAC.String(),
		string(ips),
		string(names),
		d.LastSeen.Unix(),
	)
}

func (s *SQLiteStore) All() []*model.Device {
	rows, _ := s.db.Query(`SELECT id FROM devices`)
	defer rows.Close()

	var out []*model.Device
	for rows.Next() {
		var id string
		rows.Scan(&id)
		if d, ok := s.Get(id); ok {
			out = append(out, d)
		}
	}
	return out
}
