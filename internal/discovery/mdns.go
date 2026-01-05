package discovery

import (
	"context"
	"net"
	"net/netip"
	"strings"
	"time"
)

type MDNSDiscoverer struct{}

func NewMDNS() *MDNSDiscoverer {
	return &MDNSDiscoverer{}
}

func (m *MDNSDiscoverer) Name() string {
	return "mdns"
}

func (m *MDNSDiscoverer) Discover(ctx context.Context, out chan<- Sighting) error {
	addr, err := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		return err
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 9000)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil || n == 0 {
				continue
			}

			name := extractDNSName(buf[:n])
			if name == "" {
				continue
			}

			ip, ok := netip.AddrFromSlice(src.IP)
			if !ok {
				continue
			}

			out <- Sighting{
				IP:       ip,
				Hostname: name,
				Source:   m.Name(),
				SeenAt:   time.Now(),
			}
		}
	}
}

func extractDNSName(pkt []byte) string {
	s := string(pkt)
	if !strings.Contains(s, ".local") {
		return ""
	}

	parts := strings.Split(s, ".local.")
	if len(parts) == 0 {
		return ""
	}

	return strings.Trim(parts[0], "\x00")
}
