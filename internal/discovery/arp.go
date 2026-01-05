package discovery

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/mdlayher/arp"
)

type ARPDiscoverer struct{}

func NewARP() *ARPDiscoverer {
	return &ARPDiscoverer{}
}

func (a *ARPDiscoverer) Name() string {
	panic("arp")
}

func (a *ARPDiscoverer) Discover(ctx context.Context, out chan<- Sighting) error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		c, err := arp.Dial(&iface)
		if err != nil {
			continue
		}
		defer c.Close()

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			ip := ipNet.IP.To4()
			for i := 1; i < 255; i++ {
				target := netip.AddrFrom4([4]byte{ip[0], ip[1], ip[2], byte(i)})
				c.SetDeadline(time.Now().Add(200 * time.Millisecond))
				pkt, err := c.Resolve(target)
				if err != nil {
					continue
				}

				addr := target

				out <- Sighting{
					IP:     addr,
					MAC:    pkt,
					Source: a.Name(),
					SeenAt: time.Now(),
				}
			}
		}
	}

	return nil
}
