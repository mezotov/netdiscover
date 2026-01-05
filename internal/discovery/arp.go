package discovery

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/mdlayher/arp"
	"github.com/mezotov/netdiscover/internal/security"
	"github.com/mezotov/netdiscover/internal/worker"
)

type ARPDiscoverer struct{}

func NewARP() *ARPDiscoverer {
	return &ARPDiscoverer{}
}

func (a *ARPDiscoverer) Name() string {
	return "arp"
}

func (a *ARPDiscoverer) Discover(ctx context.Context, out chan<- Sighting) error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	if err := security.RequireRoot("ARP Discovery"); err != nil {
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

		go func() {
			<-ctx.Done()
			c.Close()
		}()

		pool := worker.New[netip.Addr](64, func(target netip.Addr) {
			c.SetDeadline(time.Now().Add(100 * time.Millisecond))
			pkt, err := c.Resolve(target)
			if err != nil {
				return
			}

			out <- Sighting{
				IP:     target,
				MAC:    pkt,
				Source: a.Name(),
				SeenAt: time.Now(),
			}
		})

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// We only want IPv4
			if ipNet.IP.To4() == nil {
				continue
			}

			// Convert to netip
			prefix, err := netip.ParsePrefix(ipNet.String())
			if err != nil {
				continue
			}

			// Iterate over the subnet
			for ip := prefix.Addr(); prefix.Contains(ip); ip = ip.Next() {
				// Skip network and broadcast if possible, or just scan everything
				// Simple iteration:
				if ip.Is4() {
					pool.Submit(ip)
				}
			}
		}

		pool.Wait()
		pool.Close()
	}

	return nil
}
