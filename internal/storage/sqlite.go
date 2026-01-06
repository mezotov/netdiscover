package storage

import (
	"database/sql"
	"fmt"
	"netdis/internal/model"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &Storage{db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS scan_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		network TEXT NOT NULL,
		interface TEXT NOT NULL,
		duration TEXT NOT NULL,
		total_devices INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scan_id INTEGER,
		ip TEXT NOT NULL,
		mac TEXT,
		hostname TEXT,
		manufacturer TEXT,
		status TEXT NOT NULL,
		first_seen DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (scan_id) REFERENCES scan_results(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		port INTEGER NOT NULL,
		protocol TEXT NOT NULL,
		service TEXT NOT NULL,
		state TEXT NOT NULL,
		detected_at DATETIME NOT NULL,
		FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_devices_ip ON devices(ip);
	CREATE INDEX IF NOT EXISTS idx_devices_mac ON devices(mac);
	CREATE INDEX IF NOT EXISTS idx_devices_manufacturer ON devices(manufacturer);
	CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen);
	CREATE INDEX IF NOT EXISTS idx_scan_results_timestamp ON scan_results(timestamp);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *Storage) SaveScanResult(result model.ScanResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO scan_results (timestamp, network, interface, duration, total_devices)
		VALUES (?, ?, ?, ?, ?)
	`, result.TimeStamp, result.Network, result.Interface, result.Duration, result.Total)
	if err != nil {
		return err
	}

	scanID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	for _, device := range result.Devices {
		res, err := tx.Exec(`
			INSERT INTO devices (scan_id, ip, mac, hostname, manufacturer, status, first_seen, last_seen)
			VALUES (?, ?, ?, ?, ?, ?, ?)
	`, scanID, device.IP, device.MAC, device.Hostname, device.Manufacturer, device.Status, device.FirstSeen, device.LastSeen)
		if err != nil {
			return err
		}

		deviceID, err := res.LastInsertId()
		if err != nil {
			return err
		}

		for _, service := range device.Services {
			_, err := tx.Exec(`
				INSERT INTO services (device_id, port, protocol, service, state, detected_at)
				VALUES (?, ?, ?, ?, ?, ?)
		`, deviceID, service.Port, service.Protocol, service.Service, service.State, service.DetectedAt)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *Storage) SearchDevices(filter model.SearchFilter) ([]model.Device, error) {
	query := `SELECT DISTINCT d.id, d.ip, d.mac, d.hostname, d.manufacturer, d.status, d.first_seen, d.last_seen FROM devices d WHERE 1=1`
	args := []interface{}{filter.IP, filter.MAC}

	if filter.IP != "" {
		query += " AND d.ip LIKE ?"
		args = append(args, "%"+filter.IP+"%")
	}

	if filter.MAC != "" {
		query += " AND d.mac LIKE ?"
		args = append(args, "%"+strings.ToUpper(filter.MAC)+"%")
	}

	if filter.Hostname != "" {
		query += " AND d.hostname LIKE ?"
		args = append(args, "%"+filter.Hostname+"%")
	}

	if filter.Manufacturer != "" {
		query += " AND d.manufacturer LIKE ?"
		args = append(args, "%"+filter.Manufacturer+"%")
	}

	if filter.Status != "" {
		query += " AND d.status = ?"
		args = append(args, filter.Status)
	}

	if filter.FromDate != nil {
		query += " AND d.last_seen >= ?"
		args = append(args, filter.FromDate)
	}

	if filter.ToDate != nil {
		query += " AND d.last_seen <= ?"
		args = append(args, filter.ToDate)
	}

	query += " ORDER BY d.last_seen DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var d model.Device
		err := rows.Scan(&d.ID, &d.IP, &d.MAC, &d.Hostname, &d.Manufacturer,
			&d.Status, &d.FirstSeen, &d.LastSeen)
		if err != nil {
			return nil, err
		}

		services, err := s.getDeviceServices(d.ID)
		if err != nil {
			return nil, err
		}
		d.Services = services

		devices = append(devices, d)
	}

	return devices, rows.Err()
}

func (s *Storage) getDeviceServices(deviceID int) ([]model.Service, error) {
	rows, err := s.db.Query(`
		SELECT id, device_id, port, protocol, service, state, detected_at
		FROM services WHERE device_id = ? ORDER BY port
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []model.Service
	for rows.Next() {
		var svc model.Service
		err := rows.Scan(&svc.ID, &svc.DeviceID, &svc.Port, &svc.Protocol, &svc.Service, &svc.State, &svc.DetectedAt)
		if err != nil {
			return nil, err
		}
		services = append(services, svc)
	}

	return services, rows.Err()
}

func (s *Storage) GetScanHistory(limit int) ([]model.ScanResult, error) {
	rows, err := s.db.Query(`
		SELECT id, timestamp, network, interface, duration, total_devices
		FROM scan_results ORDER BY timestamp DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ScanResult
	for rows.Next() {
		var r model.ScanResult
		err := rows.Scan(&r.ID, &r.TimeStamp, &r.Network, &r.Interface, &r.Duration, &r.Total)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Storage) PurgeOldData(retention time.Duration) (int64, error) {
	if retention == 0 {
		return 0, nil
	}

	cutoff := time.Now().Add(-retention)

	result, err := s.db.Exec(`
		DELETE FROM scan_results WHERE timestamp < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (s *Storage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalScans int
	err := s.db.QueryRow("SELECT COUNT(*) FROM scan_results").Scan(&totalScans)
	if err != nil {
		return nil, err
	}
	stats["total_scans"] = totalScans

	var totalDevices int
	err = s.db.QueryRow("SELECT COUNT(DISTINCT ip) FROM devices").Scan(&totalDevices)
	if err != nil {
		return nil, err
	}
	stats["total_unique_devices"] = totalDevices

	var latestScan time.Time
	err = s.db.QueryRow("SELECT MAX(timestamp) FROM scan_results").Scan(&latestScan)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	stats["latest_scan"] = latestScan

	var oldestScan time.Time
	err = s.db.QueryRow("SELECT MIN(timestamp) FROM scan_results").Scan(&oldestScan)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	stats["oldest_scan"] = oldestScan

	return stats, nil
}
