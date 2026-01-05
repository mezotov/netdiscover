package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/mezotov/netdiscover/internal/correlate"
	"github.com/mezotov/netdiscover/internal/discovery"
	"github.com/mezotov/netdiscover/internal/output"
	"github.com/mezotov/netdiscover/internal/store"
)

func main() {
	printBanner()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sightings := make(chan discovery.Sighting, 256)

	memStore := store.NewMemoryStore()
	corr := correlate.New(memStore)

	go corr.Run(ctx, sightings)

	discoverers := []discovery.Discoverer{
		discovery.NewARP(),
		discovery.NewICMP(),
		discovery.NewReverseDNS(),
	}

	for _, d := range discoverers {
		go func(d discovery.Discoverer) {
			if err := d.Discover(ctx, sightings); err != nil {
				log.Printf("[%s] error: %v", d.Name(), err)
			}
		}(d)
	}

	time.Sleep(5 * time.Second)
	cancel()

	time.Sleep(500 * time.Millisecond)

	output.PrintTable(memStore.All())
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
