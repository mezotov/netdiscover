package store

import "github.com/mezotov/netdiscover/internal/model"

func Snapshot(devs []*model.Device) []*model.Device {
	out := make([]*model.Device, 0, len(devs))
	for _, d := range devs {
		copied := *d
		out = append(out, &copied)
	}

	return out
}
