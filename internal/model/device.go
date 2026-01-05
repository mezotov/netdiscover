package model

import (
	"net"
	"time"
)

type Device struct {
	ID           string
	IP           net.Addr
	MAC          net.HardwareAddr
	Hostname     string
	Manufacturer string
	Status       string
	LastSeen     time.Time
	Confidence   Confidence
}

func NewDevice(id string) *Device {
	return &Device{
		ID: id,
	}
}
