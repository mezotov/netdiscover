package model

import (
	"net"
	"net/netip"
	"time"
)

type Device struct {
	ID        string
	MAC       net.HardwareAddr
	IPs       map[netip.Addr]time.Time
	Hostnames map[string]time.Time
	Vendor    string
	LastSeen  time.Time
}

func NewDevice(id string) *Device {
	return &Device{
		ID:        id,
		IPs:       make(map[netip.Addr]time.Time),
		Hostnames: make(map[string]time.Time),
	}
}
