package network

import (
	"fmt"
	"net"
)

type InterfaceInfo struct {
	Interface *net.Interface
	IPNet     *net.IPNet
	IP        string
	Network   string
}

func GetAllInterfaces() ([]InterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []InterfaceInfo

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

			result = append(result, InterfaceInfo{
				Interface: &iface,
				IPNet:     ipNet,
				IP:        ipNet.IP.String(),
				Network:   ipNet.String(),
			})
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no active network interface found")
	}

	return result, nil
}

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
