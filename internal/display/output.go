package display

import (
	"encoding/json"
	"fmt"
	"netdis/internal/model"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

func PrintBanner() {
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

func PrintScanHistory(results []*model.ScanResult) {
	if len(results) == 0 {
		color.Yellow("⚠ No scan history found")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Timestamp", "Network", "Interface", "Duration", "Devices"})

	table.SetBorder(true)
	table.SetRowLine(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
	)

	for _, r := range results {
		table.Append([]string{
			fmt.Sprintf("%d", r.ID),
			r.TimeStamp.Format("2006-01-02 15:04:05"),
			r.Network,
			r.Interface,
			r.Duration,
			fmt.Sprintf("%d", r.Total),
		})
	}

	table.Render()
}

func PrintStats(stats map[string]interface{}) {
	fmt.Println()
	color.Cyan("📊 Database Statistics")
	color.Cyan("═══════════════════════")

	for key, value := range stats {
		label := strings.ReplaceAll(key, "-", " ")
		label = strings.ToTitle(label)
		fmt.Printf("%-25s: %v\n", label, value)
	}
	fmt.Println()
}

func ExportJSON(result *model.ScanResult, filename string) error {
	data, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		return fmt.Errorf("failed to JSON: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	color.Green("\n📄 Results exported to %s", filename)
	return nil
}

func PrintSuccess(format string, args ...interface{}) {
	color.Green(format, args...)
}

func PrintError(format string, args ...interface{}) {
	color.Red(format, args...)
}

func PrintInfo(format string, args ...interface{}) {
	color.Cyan(format, args...)
}

func PrintWarning(format string, args ...interface{}) {
	color.Yellow(format, args...)
}
