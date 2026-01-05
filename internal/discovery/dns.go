package discovery

import (
	"context"
	"net"
	"net/netip"
	"time"
)

type ReverseDNSDiscoverer struct{}

func NewReverseDNS() *ReverseDNSDiscoverer {
	return &ReverseDNSDiscoverer{}
}

func (r *ReverseDNSDiscoverer) Name() string {
	return "rdns"
}

func (r *ReverseDNSDiscoverer) Discover(ctx context.Context, out chan<- Sighting) error {
	for i := 1; i < 255; i++ {
		ip := net.IPv4(192, 168, 0, byte(i))
		names, err := net.LookupAddr(ip.String())
		if err != nil || len(names) == 0 {
			continue
		}

		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			continue
		}

		out <- Sighting{
			IP:       addr,
			Hostname: names[0],
			Source:   r.Name(),
			SeenAt:   time.Now(),
		}
	}

	return nil
}
