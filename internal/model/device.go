package model

import (
	"time"
)

type Device struct {
	ID           string     `json:"id"`
	IP           string     `json:"ip"`
	MAC          string     `json:"mac"`
	Hostname     string     `json:"hostname"`
	Manufacturer string     `json:"manufacturer"`
	Status       string     `json:"status"`
	Services     []Service  `json:"services"`
	FirstSeen    time.Time  `json:"first_seen"`
	LastSeen     time.Time  `json:"last_seen"`
	Confidence   Confidence `json:"confidence"`
}

func NewDevice(id string) *Device {
	return &Device{
		ID: id,
	}
}
