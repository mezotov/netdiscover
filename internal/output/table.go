package output

import (
	"fmt"
	"strings"

	"github.com/mezotov/netdiscover/internal/model"
)

func PrintTable(devices []*model.Device) {
	fmt.Printf("%-40s %-18s %-25s %-10s %-10s\n", "ID", "MAC", "VENDOR", "CONF", "HOSTNAMES")
	for _, d := range devices {
		fmt.Printf(
			"%-40s %-18s %-25s %-10d %-10s\n",
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
