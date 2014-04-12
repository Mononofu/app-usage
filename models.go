package usage

import (
	"time"
)

type HourlyUsage struct {
	At     time.Time // Hour of the usage
	Events []Usage   `datastore:",noindex"`
}

type Usage struct {
	At           time.Time     `datastore:",noindex"`
	Focused      App           `datastore:",noindex"`
	LastActivity time.Duration `datastore:",noindex"`
	Hostname     string        `datastore:",noindex"`
}

type App struct {
	WindowTitle string `datastore:",noindex"`
	Process     string `datastore:",noindex"`
}
