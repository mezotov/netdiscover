package cmd

import (
	"os"
	"path/filepath"
	"time"

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
	Run: runSearch
}

var historicCmd = &cobra.Command{
	Use:   "history",
	Short: "Show scan history",
	Long: "Display recent scan results from the database",
	Run: runHistory,
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database statistics",
	Long: "Display statistics about stored scans and devices",
	Run: runStats,
}

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge old scan data",
	Long:  "Remove scan data older than the retention policy",
	Run: runPurge,
}

var (
	searchIP string
	searchMAC string
	searchHostname string
	searchManufacturer string
	searchLimit int
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
