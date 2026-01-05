package network

import (
	"fmt"
	"net"
)

func GetLocalNetwork() (*net.Interface, *net.IPNet, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
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

			return &iface, ipNet, nil
		}
	}

	return nil, nil, fmt.Errorf("no active network interface found")
}
