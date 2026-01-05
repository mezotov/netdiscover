package config

import "time"

type Config struct {
	JSONExport      string
	DetectServices  bool
	PeriodicScan    bool
	ScanInterval    time.Duration
	ShowChangesOnly bool
}
