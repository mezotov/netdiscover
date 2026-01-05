package store

import "github.com/mezotov/netdiscover/internal/model"

type Store interface {
	Get(id string) (*model.Device, bool)
	Save(d *model.Device)
	All() []*model.Device
}
