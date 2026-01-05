package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"netdis/internal/model"
	"netdis/internal/vendors"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type Scanner struct {
	network *net.IPNet
	timeout time.Duration
	workers int
}

func main() {
	printBanner()

	iface, network, err := getLocalNetwork()
	if err != nil {
		color.Red("✗ Error: %v", err)
		os.Exit(1)
	}

	color.Cyan("📡 Scanning network: %s", network.String())
	color.Cyan("🔌 Interface: %s", iface.Name)
	fmt.Println()

	scanner := &Scanner{
		network: network,
		timeout: 1 * time.Second,
		workers: 100,
	}

	devices := scanner.Scan()

	if len(devices) == 0 {
		color.Yellow("⚠ No devices found")
		return
	}

	printDevices(devices)
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

func getLocalNetwork() (*net.Interface, *net.IPNet, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			return &iface, ipNet, nil
		}
	}

	return nil, nil, fmt.Errorf("no active network interface found")
}

func (s *Scanner) Scan() []*model.Device {
	ips := s.getIPRange()

	color.Green("🔍 Scanning %d hosts...\n", len(ips))

	start := time.Now()
	devices := s.scanIPs(ips)
	duration := time.Since(start)

	color.Green("\n✓ Scan completed in %v", duration)
	color.Green("✓ Found %d active devices\n", len(devices))

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
	if !ping(ctx, ip.String(), s.timeout) {
		return &model.Device{}, false
	}

	device := &model.Device{
		IP:     ip,
		Status: "Active",
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

func printDevices(devices []*model.Device) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"IP Address", "MAC Address", "Hostname", "Manufacturer", "Status"})

	table.SetBorder(true)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
	)
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgYellowColor},
		tablewriter.Colors{tablewriter.FgBlueColor},
		tablewriter.Colors{tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.FgMagentaColor},
		tablewriter.Colors{tablewriter.FgGreenColor},
	)

	for _, device := range devices {
		hostname := device.Hostname
		if hostname == "" {
			hostname = "-"
		}

		mac := device.MAC
		if mac == "" {
			mac = "-"
		}

		table.Append([]string{
			device.IP,
			mac,
			hostname,
			device.Manufacturer,
			device.Status,
		})
	}

	table.Render()
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
