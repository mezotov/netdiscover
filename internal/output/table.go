package output

import (
	"fmt"
	"netdis/internal/model"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func PrintDevices(devices []*model.Device, showServices bool) {
	table := tablewriter.NewWriter(os.Stdout)
	if showServices {
		table.SetHeader([]string{"IP Address", "MAC Address", "Hostname", "Manufacturer", "Open Ports", "Status"})
	} else {
		table.SetHeader([]string{"IP Address", "MAC Address", "Hostname", "Manufacturer", "Status"})
	}

	table.SetBorder(true)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	colors := []tablewriter.Colors{
		{tablewriter.Bold, tablewriter.FgCyanColor},
		{tablewriter.Bold, tablewriter.FgCyanColor},
		{tablewriter.Bold, tablewriter.FgCyanColor},
		{tablewriter.Bold, tablewriter.FgCyanColor},
		{tablewriter.Bold, tablewriter.FgCyanColor},
	}

	if showServices {
		colors = append(colors, tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	}

	table.SetHeaderColor(colors...)

	colColors := []tablewriter.Colors{
		{tablewriter.FgYellowColor},
		{tablewriter.FgBlueColor},
		{tablewriter.FgGreenColor},
		{tablewriter.FgMagentaColor},
	}

	if showServices {
		colColors = append(colColors, tablewriter.Colors{tablewriter.FgCyanColor})
	}

	colColors = append(colColors, tablewriter.Colors{tablewriter.FgGreenColor})
	table.SetColumnColor(colColors...)

	for _, device := range devices {
		hostname := device.Hostname
		if hostname == "" {
			hostname = "-"
		}

		mac := device.MAC
		if mac == "" {
			mac = "-"
		}

		row := []string{
			device.IP,
			mac,
			hostname,
			device.Manufacturer,
		}

		if showServices {
			ports := formatPorts(device.Services)
			row = append(row, ports)
		}

		row = append(row, device.Status)
		table.Append(row)
	}

	table.Render()
}

func formatPorts(services []model.Service) string {
	if len(services) == 0 {
		return "-"
	}

	var ports []string
	for _, s := range services {
		ports = append(ports, fmt.Sprintf("%d/%s", s.Port, s.Service))
	}

	result := strings.Join(ports, ", ")
	if len(result) < 60 {
		return result[:57] + "..."
	}

	return result
}
