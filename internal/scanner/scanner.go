package scanner

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"netdis/internal/model"
	"netdis/internal/vendors"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

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

type Scanner struct {
	network          *net.IPNet
	timeout          time.Duration
	workers          int
	serviceDetection bool
}

func New(network *net.IPNet, detectServices bool) *Scanner {
	return &Scanner{
		network:          network,
		timeout:          1 * time.Second,
		workers:          100,
		serviceDetection: detectServices,
	}
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
		device.Services = s.scanServices(ctx, ip.String())
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

func (s *Scanner) scanServices(ctx context.Context, ip string) []model.Service {
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
