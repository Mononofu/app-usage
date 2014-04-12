package usage

import (
	"net/url"
	"strings"
	"time"

	"common"
	"models"
)

func MakeChromeUsage() Logger {
	return &chromeUsage{
		sites: make(map[string]time.Duration),
	}
}

type chromeUsage struct {
	sites map[string]time.Duration
}

func (c *chromeUsage) AddUsage(usage models.Usage) {
	host := tryGetHostname(usage.Focused.WindowTitle)
	c.sites[host] += common.LogInterval
}

func (c *chromeUsage) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"name":     "chrome",
		"children": serializeChildren(c.sites),
	}
}

func tryGetHostname(name string) string {
	if len(strings.Split(name, "-")) == 2 && strings.Contains(name, ".") {
		if !strings.HasPrefix(name, "http") {
			name = "http://" + name
		}
		parsedUrl, err := url.Parse(strings.Split(name, "-")[0])
		if err != nil {
			return name
		}
		return parsedUrl.Host
	}
	titleParts := strings.Split(name, "-")
	host := titleParts[len(titleParts)-2]
	return strings.Replace(strings.Replace(host, "www.", "", -1), ".com", "", -1)
}
