package config

import "time"

type Config struct {
	EnableARP  bool
	EnableICMP bool
	EnableDNS  bool

	ScanTimeout time.Duration
	HTTPAddr    string
}
