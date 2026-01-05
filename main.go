package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"netdis/internal/config"
	"netdis/internal/model"
	"netdis/internal/network"
	"netdis/internal/output"
	"netdis/internal/vendors"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
)

type Scanner struct {
	network          *net.IPNet
	timeout          time.Duration
	workers          int
	serviceDetection bool
}

var (
	commonPorts = []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 3306, 3389, 5432, 5900, 8080, 8443}
	serviceMap  = map[int]string{
		21:   "ftp",
		22:   "SSH",
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		80:   "HTTP",
		110:  "POP3",
		143:  "IMAP",
		443:  "HTTPS",
		445:  "SMB",
		3306: "MySQL",
		3389: "RDP",
		5432: "PostgreSQL",
		5900: "VNC",
		8080: "HTTP-Alt",
		8443: "HTTPS-Alt",
	}
)

func main() {
	cfg := parseFlags()

	printBanner()

	iface, lnet, err := network.GetLocalNetwork()
	if err != nil {
		color.Red("✗ Error: %v", err)
		os.Exit(1)
	}

	scanner := &Scanner{
		network:          lnet,
		timeout:          1 * time.Second,
		workers:          100,
		serviceDetection: cfg.DetectServices,
	}

	if cfg.PeriodicScan {
		runPeriodicScan(scanner, iface, lnet, cfg)
	} else {
		result := runSingleScan(scanner, iface, lnet)

		if cfg.JSONExport != "" {
			exportJSON(result, cfg.JSONExport)
		}
	}
}

func parseFlags() config.Config {
	cfg := config.Config{}

	flag.StringVar(&cfg.JSONExport, "json", "", "Export results to JSON file")
	flag.StringVar(&cfg.JSONExport, "j", "", "Export results to JSON file (shorthand)")
	flag.BoolVar(&cfg.DetectServices, "services", false, "Detect services on open ports")
	flag.BoolVar(&cfg.DetectServices, "s", false, "Detect services on open ports (shorthand)")
	flag.BoolVar(&cfg.PeriodicScan, "watch", false, "Continuous scanning mode")
	flag.BoolVar(&cfg.PeriodicScan, "w", false, "Continuous scanning mode (shorthand)")
	flag.DurationVar(&cfg.ScanInterval, "interval", 30*time.Second, "Scan interval for watch mode")
	flag.BoolVar(&cfg.ShowChangesOnly, "changes", false, "Show only changes in watch mode")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Network Device Discovery Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                           # Basic scan\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -s                        # Scan with service detection\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -j results.json           # Export to JSON\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -w -interval 60s          # Watch mode, scan every 60s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -w -changes               # Watch mode, show only changes\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -s -j results.json        # Full scan with JSON export\n", os.Args[0])
	}

	flag.Parse()
	return cfg
}

func printBanner() {
	banner := `
 ****     ** ******** ********** *******   **  ********   ******    *******   **      ** ******** *******  
/**/**   /**/**///// /////**/// /**////** /** **//////   **////**  **/////** /**     /**/**///// /**////** 
/**//**  /**/**          /**    /**    /**/**/**        **    //  **     //**/**     /**/**      /**   /** 
/** //** /**/*******     /**    /**    /**/**/*********/**       /**      /**//**    ** /******* /*******  
/**  //**/**/**////      /**    /**    /**/**////////**/**       /**      /** //**  **  /**////  /**///**  
/**   //****/**          /**    /**    ** /**       /**//**    **//**     **   //****   /**      /**  //** 
/**    //***/********    /**    /*******  /** ********  //******  //*******     //**    /********/**   //**
//      /// ////////     //     ///////   // ////////    //////    ///////       //     //////// //     // 

Fast • Beautiful • Production Ready 

`
	color.Cyan(banner)
}

func runSingleScan(scanner *Scanner, iface *net.Interface, lnet *net.IPNet) model.ScanResult {
	color.Cyan("📡 Scanning network: %s", lnet.String())
	color.Cyan("🔌 Interface: %s", iface.Name)
	if scanner.serviceDetection {
		color.Cyan("🔎 Service Detection: enabled")
	}
	fmt.Println()

	start := time.Now()
	devices := scanner.Scan()
	duration := time.Since(start)

	result := model.ScanResult{
		TimeStamp: time.Now(),
		Network:   lnet.String(),
		Interface: iface.Name,
		Duration:  duration.String(),
		Total:     len(devices),
		Devices:   devices,
	}

	if len(devices) == 0 {
		color.Yellow("⚠ No devices found")
		return result
	}

	color.Green("\n✓ Scan completed in %v", duration)
	color.Green("✓ Found %d active devices\n", len(devices))

	output.PrintDevices(devices, scanner.serviceDetection)

	return result
}

func runPeriodicScan(scanner *Scanner, iface *net.Interface, lnet *net.IPNet, config config.Config) {
	color.Cyan("📡 Network: %s", lnet.String())
	color.Cyan("🔌 Interface: %s", iface.Name)
	color.Cyan("⏱️  Interval: %v", config.ScanInterval)
	if config.ShowChangesOnly {
		color.Cyan("👁️ Mode: Changes only")
	}
	fmt.Println()

	previousDevices := make(map[string]model.Device)
	scanCount := 0

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(config.ScanInterval)
	defer ticker.Stop()

	runPeriodicScanIteration(scanner, &previousDevices, &scanCount, config.ShowChangesOnly, true)

	for {
		select {
		case <-ticker.C:
			runPeriodicScanIteration(scanner, &previousDevices, &scanCount, config.ShowChangesOnly, false)
		case <-sigChan:
			color.Yellow("\n\n⏹️ Stopping periodic scan...")
			color.Green("📊 Total scans performed: %d", scanCount)
			return
		}
	}
}

func runPeriodicScanIteration(scanner *Scanner, previousDevices *map[string]model.Device, scanCount *int, showChangesOnly bool, isFirst bool) {
	*scanCount++

	timestamp := time.Now().Format("15:04:05")
	color.HiWhite("\n═══════════════════════════════════════")
	color.HiCyan("🔄 Scan #%d - %s", *scanCount, timestamp)
	color.HiWhite("═══════════════════════════════════════")

	devices := scanner.Scan()

	added, removed, changed := detectChanges(*previousDevices, devices)

	if showChangesOnly && !isFirst {
		if len(added) == 0 && len(removed) == 0 && len(changed) == 0 {
			color.Green("✓ No changes detected (%d devices)", len(devices))
		} else {
			printChanges(added, removed, changed)
		}
	} else {
		color.Green("\n✓ Found %d active devices", len(devices))
		if len(added) > 0 || len(removed) > 0 || len(changed) > 0 {
			printChanges(added, removed, changed)
		}
		fmt.Println()
		output.PrintDevices(devices, scanner.serviceDetection)
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

func exportJSON(result model.ScanResult, filename string) {
	data, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		color.Red("✗ Error creating JSON: %v", err)
		return
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		color.Red("✗ Error writing file: %v", err)
		return
	}

	color.Green("\n📄 Results exported to %s", filename)
}

func (s *Scanner) Scan() []*model.Device {
	ips := s.getIPRange()

	devices := s.scanIPs(ips)

	return devices
}

func (s *Scanner) getIPRange() []net.IP {
	var ips []net.IP

	ip := s.network.IP.Mask(s.network.Mask)

	for ip := ip.Mask(s.network.Mask); s.network.Contains(ip); inc(ip) {
		if ip[3] == 0 || ip[3] == 255 {
			continue
		}
		ipCopy := make(net.IP, len(ip))
		copy(ipCopy, ip)
		ips = append(ips, ipCopy)
	}

	return ips
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func (s *Scanner) scanIPs(ips []net.IP) []*model.Device {
	var wg sync.WaitGroup
	deviceChan := make(chan *model.Device, len(ips))
	semaphore := make(chan struct{}, s.workers)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, ip := range ips {
		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()

				if device, found := s.scanHost(ctx, ip); found {
					deviceChan <- device
				}
			case <-ctx.Done():
				return
			}
		}(ip)
	}

	go func() {
		wg.Wait()
		close(deviceChan)
	}()

	var devices []*model.Device
	for device := range deviceChan {
		devices = append(devices, device)
	}

	sort.Slice(devices, func(i, j int) bool {
		return ipToInt(devices[i].IP) < ipToInt(devices[j].IP)
	})

	return devices
}

func (s *Scanner) scanHost(ctx context.Context, ip net.IP) (*model.Device, bool) {
	now := time.Now()
	device := &model.Device{
		IP:        ip.String(),
		Status:    "Active",
		FirstSeen: now,
		LastSeen:  now,
	}

	if s.serviceDetection {
		device.Services = s.detectServices(ctx, ip.String())
		if len(device.Services) == 0 {
			return &model.Device{}, false
		}
	} else {
		if !ping(ctx, ip.String(), s.timeout) {
			return &model.Device{}, false
		}
	}

	oui, _ := vendors.Load("oui.txt")

	if mac := getMAC(ip.String()); mac != "" {
		device.MAC = mac
		device.Manufacturer = oui.Vendor(mac)
	}

	if hostname := getHostname(ip.String()); hostname != "" {
		device.Hostname = hostname
	}

	return device, true
}

func (s *Scanner) detectServices(ctx context.Context, ip string) []model.Service {
	var services []model.Service
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, port := range commonPorts {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()

			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), s.timeout)
			if err != nil {
				conn.Close()

				serviceName := serviceMap[port]
				if serviceName == "" {
					serviceName = "Unknown"
				}

				mu.Lock()
				services = append(services, model.Service{
					Port:     port,
					Protocol: "tcp",
					Service:  serviceName,
					State:    "open",
				})
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	sort.Slice(services, func(i, j int) bool {
		return services[i].Port < services[j].Port
	})

	return services
}

func ping(ctx context.Context, ip string, timeout time.Duration) bool {
	ports := []string{"80", "443", "22", "445"}

	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), timeout)
		if err == nil {
			conn.Close()
			return true
		}
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", "500", ip)
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ip)
	}

	err := cmd.Run()
	return err == nil
}

func getMAC(ip string) string {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("arp", "-a", ip)
	} else {
		cmd = exec.Command("arp", "-n", ip)
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ip) {
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.Count(field, ":") == 5 || strings.Count(field, "-") == 5 {
					mac := strings.ReplaceAll(field, "-", ":")
					return strings.ToUpper(mac)
				}
			}
		}
	}

	return ""
}

func getHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}

	hostname := names[0]
	hostname = strings.TrimSuffix(hostname, ".")

	return hostname
}

func ipToInt(ip string) uint32 {
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return 0
	}
	netIP = netIP.To4()
	if netIP == nil {
		return 0
	}
	return binary.BigEndian.Uint32(netIP)
}
