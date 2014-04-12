package usage

import (
	"models"
)

type UsageLogger interface {
	AddUsage(usage models.Usage)
	Serialize() map[string]interface{}
}

func MakeUsageLogger() (UsageLogger, error) {
	l := usageLogger{}
	l.apps = make(map[string]UsageLogger)
	l.apps["chrome"] = MakeChromeUsage()
	sublime, err := MakeSublimeUsage("~/Dropbox/Programmieren/([^/]+)/.*",
		"~/Programmieren/([^/]+)/.*")
	if err != nil {
		return nil, err
	}
	l.apps["sublime_text"] = sublime
	return &l, nil
}

type usageLogger struct {
	apps map[string]UsageLogger
}

func (l *usageLogger) AddUsage(usage models.Usage) {
	if _, ok := l.apps[usage.Focused.Process]; !ok {
		l.apps[usage.Focused.Process] = MakeAppUsage(usage.Focused.Process)
	}
	l.apps[usage.Focused.Process].AddUsage(usage)
}

func (l *usageLogger) Serialize() map[string]interface{} {
	var children []interface{}
	for _, app := range l.apps {
		children = append(children, app.Serialize())
	}
	return map[string]interface{}{
		"name":     "AppUsage",
		"children": children,
	}
}
