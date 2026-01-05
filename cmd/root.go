package cmd

import (
	"fmt"
	"net"
	"netdis/internal/display"
	"netdis/internal/model"
	"netdis/internal/network"
	"netdis/internal/scanner"
	"netdis/internal/storage"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	dbPath          string
	jsonExport      string
	detectServices  bool
	periodicScan    bool
	scanInterval    time.Duration
	retentionPolicy string
	showChangesOnly bool
)

var rootCmd = &cobra.Command{
	Use:   "netdiscover",
	Short: "Network Device Discovery Tool",
	Long: `A fast and beautiful CLI tool for discovering devices on your local network.
		Features include service detection, SQLite storage, search capabilities, and more.`,
	Run: runScan,
}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search stored scan results",
	Long:  "Search through stored scan results using various filters",
	Run:   runSearch,
}

var historicCmd = &cobra.Command{
	Use:   "history",
	Short: "Show scan history",
	Long:  "Display recent scan results from the database",
	Run:   runHistory,
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database statistics",
	Long:  "Display statistics about stored scans and devices",
	Run:   runStats,
}

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge old scan data",
	Long:  "Remove scan data older than the retention policy",
	Run:   runPurge,
}

var (
	searchIP           string
	searchMAC          string
	searchHostname     string
	searchManufacturer string
	searchLimit        int
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	defaultDBPath := filepath.Join(home, ".netdiscover", "netdiscover.db")

	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDBPath, "Database path")
	rootCmd.Flags().StringVarP(&jsonExport, "json", "j", "", "Export results to JSON file")
	rootCmd.Flags().BoolVarP(&detectServices, "services", "s", false, "Detect services on open ports")
	rootCmd.Flags().BoolVarP(&periodicScan, "watch", "w", false, "Continuous scanning mode")
	rootCmd.Flags().DurationVar(&scanInterval, "interval", 30*time.Second, "Scan interval for watch mode")
	rootCmd.Flags().StringVarP(&retentionPolicy, "retention", "r", "", "Data retention policy (12h, 1d, 3d, 7d, 30d, forever)")
	rootCmd.Flags().BoolVar(&showChangesOnly, "changes", false, "Show only changes in watch mode")

	searchCmd.Flags().StringVar(&searchIP, "ip", "", "Filter by IP address (partial match)")
	searchCmd.Flags().StringVar(&searchMAC, "mac", "", "Filter by MAC address (partial match)")
	searchCmd.Flags().StringVar(&searchHostname, "hostname", "", "Filter by hostname (partial match)")
	searchCmd.Flags().StringVar(&searchManufacturer, "vendor", "", "Filter by manufacturer/vendor (partial match)")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 50, "Maximum number of results")

	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(historicCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(purgeCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func runScan(cmd *cobra.Command, args []string) {
	display.PrintBanner()

	store, err := initStorage()
	if err != nil {
		display.PrintError("✗ Failed to initialize storage", err)
		os.Exit(1)
	}
	defer store.Close()

	if retentionPolicy != "forever" {
		policy := model.RetentionPolicy(retentionPolicy)
		duration, err := model.ParseRetention(policy)
		if err == nil && duration > 0 {
			deleted, err := store.PurgeOldData(duration)
			if err == nil && deleted > 0 {
				display.PrintInfo("🗑️ Purged %d old scan(s) based on %s retention policy\n", deleted, retentionPolicy)
			}
		}
	}

	iface, lnet, err := network.GetLocalNetwork()
	if err != nil {
		display.PrintError("✗ Error: %v", err)
		os.Exit(1)
	}

	if periodicScan {
		runPeriodicMode(store, iface, lnet)
	} else {
		runSingleScanMode(store, iface, lnet)
	}
}

func runSingleScanMode(store *storage.Storage, iface *net.Interface, lnet *net.IPNet) {
	color.Cyan("📡 Scanning network: %s", lnet.String())
	color.Cyan("🔌 Interface: %s", iface.Name)
	if detectServices {
		color.Cyan("🔎 Service Detection: enabled")
	}
	fmt.Println()

	s := scanner.New(lnet, detectServices)
	start := time.Now()
	devices := s.Scan()
	duration := time.Since(start)

	result := model.ScanResult{
		TimeStamp: time.Now(),
		Network:   lnet.String(),
		Interface: iface.Name,
		Duration:  duration.String(),
		Total:     len(devices),
		Devices:   devices,
	}

	if err := store.SaveScanResult(&result); err != nil {
		display.PrintWarning("⚠ Failed to save results to database: %v", err)
	} else {
		display.PrintSuccess("✓ Results saved to database")
	}

	if len(devices) == 0 {
		display.PrintWarning("⚠ No devices found")
		return
	}

	display.PrintSuccess("\n✓ Scan completed in %v", duration)
	display.PrintSuccess("✓ Found %d active devices\n", len(devices))

	display.PrintDevices(devices, detectServices)

	if jsonExport != "" {
		if err := display.ExportJSON(&result, jsonExport); err != nil {
			display.PrintError("✗ Failed to export to JSON: %v", err)
		}
	}
}

func runPeriodicMode(store *storage.Storage, iface *net.Interface, lnet *net.IPNet) {
	display.PrintInfo("📡 Network: %s", lnet.String())
	display.PrintInfo("🔌 Interface: %s", iface.Name)
	display.PrintInfo("⏱️  Interval: %v", scanInterval)
	if showChangesOnly {
		display.PrintInfo("👁️ Mode: Changes only")
	}
	fmt.Println()

	previousDevices := make(map[string]model.Device)
	scanCount := 0

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	runPeriodicScanIteration(store, iface, lnet, &previousDevices, &scanCount, true)

	for {
		select {
		case <-ticker.C:
			runPeriodicScanIteration(store, iface, lnet, &previousDevices, &scanCount, false)
		case <-sigChan:
			display.PrintWarning("\n\n⏹️ Stopping periodic scan...")
			display.PrintSuccess("📊 Total scans performed: %d", scanCount)
			return
		}
	}
}

func runPeriodicScanIteration(store *storage.Storage, iface *net.Interface, lnet *net.IPNet, previousDevices *map[string]model.Device, scanCount *int, isFirst bool) {
	*scanCount++

	timestamp := time.Now().Format("15:04:05")
	color.HiWhite("\n═══════════════════════════════════════")
	color.HiCyan("🔄 Scan #%d - %s", *scanCount, timestamp)
	color.HiWhite("═══════════════════════════════════════")

	s := scanner.New(lnet, detectServices)
	start := time.Now()
	devices := s.Scan()
	duration := time.Since(start)

	result := model.ScanResult{
		TimeStamp: time.Now(),
		Network:   lnet.String(),
		Interface: iface.Name,
		Duration:  duration.String(),
		Total:     len(devices),
		Devices:   devices,
	}

	if err := store.SaveScanResult(&result); err != nil {
		display.PrintWarning("⚠ Failed to save scan: %v", err)
	}

	added, removed, changed := detectChanges(*previousDevices, devices)

	if showChangesOnly && !isFirst {
		if len(added) == 0 && len(removed) == 0 && len(changed) == 0 {
			display.PrintSuccess("✓ No changes detected (%d devices)", len(devices))
		} else {
			printChanges(added, removed, changed)
		}
	} else {
		display.PrintSuccess("\n✓ Found %d active devices", len(devices))
		if len(added) > 0 || len(removed) > 0 || len(changed) > 0 {
			printChanges(added, removed, changed)
		}
		fmt.Println()
		display.PrintDevices(devices, detectServices)
	}

	*previousDevices = make(map[string]model.Device)
	for _, d := range devices {
		(*previousDevices)[d.IP] = *d
	}
}

func detectChanges(previous map[string]model.Device, current []*model.Device) (added, removed, changed []*model.Device) {
	currentMap := make(map[string]model.Device)
	for _, d := range current {
		currentMap[d.IP] = *d
	}

	for _, curr := range current {
		prev, existed := previous[curr.IP]
		if !existed {
			added = append(added, curr)
		} else if deviceChanged(prev, *curr) {
			changed = append(changed, curr)
		}
	}

	for ip, prev := range previous {
		if _, exists := currentMap[ip]; !exists {
			removed = append(removed, &prev)
		}
	}
	return
}

func deviceChanged(prev, curr model.Device) bool {
	return prev.MAC != curr.MAC || prev.Hostname != curr.Hostname || len(prev.Services) != len(curr.Services)
}

func printChanges(added, removed, changed []*model.Device) {
	if len(added) > 0 {
		color.Green("\n➕ New devices: %d", len(added))
		for _, d := range added {
			color.Green("   • %s -%s (%s)", d.IP, d.Hostname, d.MAC)
		}
	}

	if len(removed) > 0 {
		color.Red("\n➖ Removed devices: %d", len(removed))
		for _, d := range removed {
			color.Red("   • %s - %s (%s)", d.IP, d.Hostname, d.MAC)
		}
	}

	if len(changed) > 0 {
		color.Yellow("\n🔄 Changed devices %d", len(changed))
		for _, d := range changed {
			color.Yellow("   • %s - %s (%s)", d.IP, d.Hostname, d.MAC)
		}
	}
}

func runSearch(cmd *cobra.Command, args []string) {
	store, err := initStorage()
	if err != nil {
		display.PrintError("✗ Failed to initialize storage: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	filter := model.SearchFilter{
		IP:           searchIP,
		MAC:          searchMAC,
		Hostname:     searchHostname,
		Manufacturer: searchManufacturer,
		Limit:        searchLimit,
	}

	devices, err := store.SearchDevices(filter)
	if err != nil {
		display.PrintError("✗ Search failed: %v", err)
		os.Exit(1)
	}

	display.PrintBanner()
	display.PrintInfo("🔍 Search results\n")
	display.PrintDevices(devices, true)
}

func runHistory(cmd *cobra.Command, args []string) {
	store, err := initStorage()
	if err != nil {
		display.PrintError("✗ Failed to initialize storage: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	results, err := store.GetScanHistory(20)
	if err != nil {
		display.PrintError("✗ Failed to retrieve history: %v", err)
		os.Exit(1)
	}

	display.PrintBanner()
	display.PrintInfo("📜 Scan History (Last 20 scans)\n")
	display.PrintScanHistory(results)
}

func runStats(cmd *cobra.Command, args []string) {
	store, err := initStorage()
	if err != nil {
		display.PrintError("✗ Failed to initialize storage: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	stats, err := store.GetStats()
	if err != nil {
		display.PrintError("✗ Failed to retrieve stats: %v", err)
		os.Exit(1)
	}

	display.PrintBanner()
	display.PrintStats(stats)
}

func runPurge(cmd *cobra.Command, args []string) {
	store, err := initStorage()
	if err != nil {
		display.PrintError("✗ Failed to initialize storage: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	if retentionPolicy == "forever" {
		display.PrintWarning("⚠ Retention policy is set to 'forever', no data will be purged")
		return
	}

	policy := model.RetentionPolicy(retentionPolicy)
	duration, err := model.ParseRetention(policy)
	if err != nil {
		display.PrintError("✗ Invalid retention policy: %v", err)
		os.Exit(1)
	}

	deleted, err := store.PurgeOldData(duration)
	if err != nil {
		display.PrintError("✗ Purge failed: %v", err)
		os.Exit(1)
	}

	display.PrintBanner()
	display.PrintSuccess("🗑️ Purged %d scan(s) older than %s", deleted, retentionPolicy)
}

func initStorage() (*storage.Storage, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return storage.New(dbPath)
}
