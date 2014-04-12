package usage

import (
	"regexp"
	"time"

	"common"
	"models"
)

// patterns is a list of regular expressions that capture the project name from the window title,
// e.g. "~/Dropbox/Programmieren/([^/]).*"
func MakeSublimeUsage(patterns ...string) (UsageLogger, error) {
	var rs []*regexp.Regexp
	for _, pattern := range patterns {
		r, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}

	return &sublimeUsage{
		projects: make(map[string]time.Duration),
		patterns: rs,
	}, nil
}

type sublimeUsage struct {
	projects map[string]time.Duration
	patterns []*regexp.Regexp
}

func (s *sublimeUsage) AddUsage(usage models.Usage) {
	title := usage.Focused.WindowTitle
	project := "misc"
	for _, pattern := range s.patterns {
		matches := pattern.FindStringSubmatch(title)
		if len(matches) >= 2 {
			project = matches[1]
			break
		}
	}
	s.projects[project] += common.LogInterval
}

func (s *sublimeUsage) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"name":     "sublime-text",
		"children": serializeChildren(s.projects),
	}
}

func serializeChildren(children map[string]time.Duration) []interface{} {
	var data []interface{}
	for name, length := range children {
		data = append(data, map[string]interface{}{
			"name": name,
			"size": int64(length.Seconds()),
		})
	}
	return data
}
