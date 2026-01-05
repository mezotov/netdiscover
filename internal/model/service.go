package model

import "time"

type Service struct {
	ID         int       `json:"id" db:"id"`
	DeviceID   int       `json:"device_id" db:"device_id"`
	Port       int       `json:"port" db:"port"`
	Protocol   string    `json:"protocol" db:"protocol"`
	Service    string    `json:"service" db:"service"`
	State      string    `json:"state" db:"state"`
	DetectedAt time.Time `json:"detected_at" db:"detected_at"`
}
