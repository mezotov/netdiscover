package model

import (
	"time"
)

type Device struct {
	ID           int        `json:"id" db:"id"`
	IP           string     `json:"ip" db:"ip"`
	MAC          string     `json:"mac" db:"mac"`
	Hostname     string     `json:"hostname" db:"hostname"`
	Manufacturer string     `json:"manufacturer" db:"manufacturer"`
	Status       string     `json:"status" db:"status"`
	Services     []Service  `json:"services" db:"-"`
	FirstSeen    time.Time  `json:"first_seen" db:"first_seen"`
	LastSeen     time.Time  `json:"last_seen" db:"last_seen"`
	Confidence   Confidence `json:"confidence" db:"confidence"`
}

type ScanResult struct {
	ID        int       `json:"id" db:"id"`
	TimeStamp time.Time `json:"timestamp" db:"timestamp"`
	Network   string    `json:"network" db:"network"`
	Interface string    `json:"interface" db:"interface"`
	Duration  string    `json:"duration" db:"duration"`
	Total     int       `json:"total_devices" db:"total_devices"`
	Devices   []*Device `json:"devices" db:"-"`
}

type SearchFilter struct {
	IP           string
	MAC          string
	Hostname     string
	Manufacturer string
	Status       string
	FromDate     *time.Time
	ToDate       *time.Time
	Limit        int
}

type RetentionPolicy string

const (
	Retention12Hours RetentionPolicy = "12h"
	Retention1Day    RetentionPolicy = "1d"
	Retention3Days   RetentionPolicy = "3d"
	Retention7Days   RetentionPolicy = "7d"
	Retention30Days  RetentionPolicy = "30d"
	RetentionForever RetentionPolicy = "forever"
)

func ParseRetention(policy RetentionPolicy) (time.Duration, error) {
	switch policy {
	case Retention12Hours:
		return 12 * time.Hour, nil
	case Retention1Day:
		return 24 * time.Hour, nil
	case Retention3Days:
		return 3 * 24 * time.Hour, nil
	case Retention7Days:
		return 7 * 24 * time.Hour, nil
	case Retention30Days:
		return 30 * 24 * time.Hour, nil
	case RetentionForever:
		return 0, nil
	default:
		return 0, nil
	}
}
