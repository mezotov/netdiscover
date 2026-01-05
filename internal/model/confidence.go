package model

type Confidence struct {
	Score int
}

func (c *Confidence) Add(source string) {
	switch source {
	case "arp":
		c.Score += 40
	case "mdns":
		c.Score += 25
	case "dns":
		c.Score += 15
	case "icmp":
		c.Score += 10
	}

	if c.Score > 100 {
		c.Score = 100
	}
}
