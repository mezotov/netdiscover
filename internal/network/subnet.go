package network

import (
	"net"
	"net/netip"
)

func LocalSubnets() ([]netip.Prefix, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var subnets []netip.Prefix

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			ip, ok := netip.AddrFromSlice(ipNet.IP)
			if !ok {
				continue
			}

			ones, _ := ipNet.Mask.Size()
			subnets = append(subnets, netip.PrefixFrom(ip, ones))
		}
	}

	return subnets, nil
}
