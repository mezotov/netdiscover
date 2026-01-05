package output

import (
	"encoding/json"
	"os"

	"github.com/mezotov/netdiscover/internal/model"
)

func PrintJSON(devices []*model.Device) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(devices)
}
