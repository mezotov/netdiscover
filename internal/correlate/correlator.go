package correlate

import (
	"context"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/mezotov/netdiscover/internal/discovery"
	"github.com/mezotov/netdiscover/internal/model"
	"github.com/mezotov/netdiscover/internal/store"
)

type Correlator struct {
	store store.Store
}

func New(s store.Store) *Correlator {
	return &Correlator{store: s}
}

func (c *Correlator) Run(ctx context.Context, in <-chan discovery.Sighting) {
	for {
		select {
		case <-ctx.Done():
			return
		case s := <-in:
			c.apply(s)
		}
	}
}

func (c *Correlator) apply(s discovery.Sighting) {
	id := deviceID(s)
	d, ok := c.store.Get(id)
	if !ok {
		d = model.NewDevice(id)
	}

	if s.MAC != nil {
		d.MAC = s.MAC
	}
	if s.IP.IsValid() {
		d.IPs[s.IP] = s.SeenAt
	}
	if s.Hostname != "" {
		d.Hostnames[s.Hostname] = s.SeenAt
	}

	d.LastSeen = time.Now()
	c.store.Save(d)
}

func deviceID(s discovery.Sighting) string {
	h := sha1.New()
	h.Write([]byte(s.IP.String()))
	return fmt.Sprintf("%x", h.Sum(nil))
}
