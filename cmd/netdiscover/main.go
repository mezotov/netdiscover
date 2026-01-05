package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/mezotov/netdiscover/internal/api"
	"github.com/mezotov/netdiscover/internal/config"
	"github.com/mezotov/netdiscover/internal/correlate"
	"github.com/mezotov/netdiscover/internal/discovery"
	"github.com/mezotov/netdiscover/internal/output"
	"github.com/mezotov/netdiscover/internal/progress"
	"github.com/mezotov/netdiscover/internal/store"
)

func main() {
	printBanner()

	cfg := config.Config{}

	jsonOut := flag.Bool("json", false, "output JSON")

	flag.BoolVar(&cfg.EnableARP, "arp", true, "enable ARP discovery")
	flag.BoolVar(&cfg.EnableICMP, "icmp", true, "enable ICMP discovery")
	flag.BoolVar(&cfg.EnableDNS, "dns", true, "enable reverse DNS discovery")
	flag.DurationVar(&cfg.ScanTimeout, "timeout", 5*time.Second, "scan timeout")
	flag.StringVar(&cfg.HTTPAddr, "http", "", "serve HTTP API")

	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sightings := make(chan discovery.Sighting, 256)

	tracker := progress.New()
	tracker.Start()

	memStore := store.NewMemoryStore()
	corr := correlate.New(memStore, tracker.Increment)

	go corr.Run(ctx, sightings)

	var discoverers []discovery.Discoverer
	if cfg.EnableARP {
		discoverers = append(discoverers, discovery.NewARP())
	}
	if cfg.EnableICMP {
		discoverers = append(discoverers, discovery.NewICMP())
	}
	if cfg.EnableDNS {
		discoverers = append(discoverers, discovery.NewReverseDNS())
	}
	if cfg.HTTPAddr != "" {
		go func() {
			server := api.New(memStore)
			log.Printf("HTTP API listening on %s", cfg.HTTPAddr)
			log.Fatal(server.Start(cfg.HTTPAddr))
		}()
	}

	for _, d := range discoverers {
		go func(d discovery.Discoverer) {
			if err := d.Discover(ctx, sightings); err != nil {
				log.Printf("[%s] error: %v", d.Name(), err)
			}
		}(d)
	}

	time.Sleep(cfg.ScanTimeout)
	cancel()
	time.Sleep(300 * time.Millisecond)

	tracker.Stop()

	if *jsonOut {
		output.PrintJSON(memStore.All())
	} else {
		output.PrintTable(memStore.All())
	}
}

func printBanner() {
	banner := `
 _   _  _____ ___________ _____ _____ _____ _____  _   _ ___________ 
| \ | ||  ___|_   _|  _  \_   _/  ___/  __ \  _  || | | |  ___| ___ \
|  \| || |__   | | | | | | | | \ --.| /  \/ | | || | | | |__ | |_/ /
| . - ||  __|  | | | | | | | |  --. \ |   | | | || | | |  __||    /
| |\  || |___  | | | |/ / _| |_/\__/ / \__/\ \_/ /\ \_/ / |___| |\ \
\_| \_/\____/  \_/ |___/  \___/\____/ \____/\___/  \___/\____/\_| \_|


`

	color.Cyan(banner)
}
