package aging

import (
	"time"

	"github.com/mezotov/netdiscover/internal/model"
)

func Expire(devs []*model.Device, maxAge time.Duration) []*model.Device {
	var out []*model.Device
	now := time.Now()

	for _, d := range devs {
		if now.Sub(d.LastSeen) <= maxAge {
			out = append(out, d)
		}
	}

	return out
}
