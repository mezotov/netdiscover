package discovery

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/mezotov/netdiscover/internal/network"
	"github.com/mezotov/netdiscover/internal/worker"
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
	subnets, err := network.LocalSubnets()
	if err != nil {
		return err
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}
	defer conn.Close()

	pool := worker.New[netip.Addr](64, func(addr netip.Addr) {
		msg := &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   1,
				Seq:  1,
				Data: []byte("ping"),
			},
		}

		b, _ := msg.Marshal(nil)
		conn.WriteTo(b, &net.IPAddr{IP: addr.AsSlice()})

		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		buf := make([]byte, 1500)

		_, peer, err := conn.ReadFrom(buf)
		if err != nil {
			return
		}

		paddr, ok := netip.AddrFromSlice(peer.(*net.IPAddr).IP)
		if !ok {
			return
		}

		out <- Sighting{
			IP:     paddr,
			Source: i.Name(),
			SeenAt: time.Now(),
		}
	})

	for _, subnet := range subnets {
		for ip := subnet.Addr(); subnet.Contains(ip); ip = ip.Next() {
			pool.Submit(ip)
		}
	}

	pool.Wait()
	pool.Close()

	return nil
}
