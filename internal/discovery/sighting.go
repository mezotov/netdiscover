package discovery

import (
	"net"
	"net/netip"
	"time"
)

type Sighting struct {
	IP       netip.Addr
	MAC      net.HardwareAddr
	Hostname string
	Source   string
	SeenAt   time.Time
}
