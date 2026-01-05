package delta

import "github.com/mezotov/netdiscover/internal/model"

type Delta struct {
	New     []*model.Device
	Updated []*model.Device
	Removed []*model.Device
}
