package discovery

import (
	"context"
)

type Discoverer interface {
	Name() string
	Discover(ctx context.Context, out chan<- Sighting) error
}
