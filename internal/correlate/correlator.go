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
	store    store.Store
	progress func(string)
	vendor   func(string) string
}

func New(s store.Store, progress func(string), vendor func(string) string) *Correlator {
	return &Correlator{store: s, progress: progress, vendor: vendor}
}

func (c *Correlator) Run(ctx context.Context, in <-chan discovery.Sighting) {
	for {
		select {
		case <-ctx.Done():
			return
		case s := <-in:
			if c.progress != nil {
				c.progress(s.Source)
			}
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
	d.Confidence.Add(s.Source)

	if d.Vendor == "" && len(d.MAC) > 0 && c.vendor != nil {
		d.Vendor = c.vendor(d.MAC.String())
	}

	c.store.Save(d)
}

func deviceID(s discovery.Sighting) string {
	if len(s.MAC) > 0 {
		return s.MAC.String()
	}

	h := sha1.New()
	h.Write([]byte(s.IP.String()))
	return fmt.Sprintf("ip:%x", h.Sum(nil))
}
