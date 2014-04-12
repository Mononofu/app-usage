package usage

import (
	"time"

	"common"
	"models"
)

func MakeAppUsage(process string) Logger {
	return &appUsage{
		process:    process,
		categories: make(map[string]time.Duration),
	}
}

type appUsage struct {
	process    string
	categories map[string]time.Duration
}

func (a *appUsage) AddUsage(usage models.Usage) {
	a.categories[usage.Focused.WindowTitle] += common.LogInterval
}

func (a *appUsage) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"name":     a.process,
		"children": serializeChildren(a.categories),
	}
}
