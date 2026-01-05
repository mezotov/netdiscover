package vendors

import (
	"bufio"
	"os"
	"strings"
)

type OUILookup struct {
	data map[string]string
}

func Load(path string) (*OUILookup, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	o := &OUILookup{data: map[string]string{}}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "(hex)") {
			continue
		}

		parts := strings.Split(line, "(hex)")
		if len(parts) != 2 {
			continue
		}

		prefix := strings.TrimSpace(strings.ReplaceAll(parts[0], "-", ":"))
		vendor := strings.TrimSpace(parts[1])
		o.data[prefix] = vendor
	}

	return o, nil
}

func (o *OUILookup) Vendor(mac string) string {
	if len(mac) < 8 {
		return ""
	}
	return o.data[strings.ToUpper(mac[:8])]
}
