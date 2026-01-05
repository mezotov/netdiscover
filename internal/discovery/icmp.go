package discovery

import (
	"context"
	"net"
	"net/netip"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type ICMPDiscoverer struct{}

func NewICMP() *ICMPDiscoverer {
	return &ICMPDiscoverer{}
}

func (i *ICMPDiscoverer) Name() string {
	return "icmp"
}

func (i *ICMPDiscoverer) Discover(ctx context.Context, out chan<- Sighting) error {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}
	defer c.Close()

	for x := 1; x < 255; x++ {
		ip := net.IPv4(192, 168, 0, byte(x))
		msg := &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   1,
				Seq:  x,
				Data: []byte("ping"),
			},
		}

		b, _ := msg.Marshal(nil)
		c.WriteTo(b, &net.IPAddr{IP: ip})

		c.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
		buf := make([]byte, 1500)
		_, peer, err := c.ReadFrom(buf)
		if err != nil {
			continue
		}

		addr, ok := netip.AddrFromSlice(peer.(*net.IPAddr).IP)
		if !ok {
			continue
		}

		out <- Sighting{
			IP:     addr,
			Source: i.Name(),
			SeenAt: time.Now(),
		}
	}

	return nil
}
