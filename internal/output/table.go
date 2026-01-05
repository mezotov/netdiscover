package output

import (
	"fmt"
	"strings"

	"github.com/mezotov/netdiscover/internal/model"
)

const ipHeader = "IP"

func PrintTable(devices []*model.Device) {
	fmt.Printf("%-20s %-45s %-18s %-25s %-10s %-10s\n", ipHeader, "ID", "MAC", "VENDOR", "CONF", "HOSTNAMES")
	for _, d := range devices {
		ip := ""
		for addr := range d.IPs {
			ip = addr.String()
			break
		}

		fmt.Printf(
			"%-20s %-45s %-18s %-25s %-10d %-10s\n",
			ip,
			d.ID,
			d.MAC,
			d.Vendor,
			d.Confidence.Score,
			strings.Join(keys(d.Hostnames), ","),
		)
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
