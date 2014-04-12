package models

import (
	"time"
)

type Piece struct {
	Start  time.Time
	Length time.Duration `datastore:",noindex"`
	Notes  []Note        `datastore:",noindex"`
}

type Note struct {
	Status            int16
	Data1             int16
	Data2             int16
	Data3             int16
	RelativeTimestamp int32
	AbsoluteTimestamp time.Time
}

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
