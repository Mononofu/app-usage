package usage

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
)

const LogInterval = time.Duration(10) * time.Second

var graphTemplate = template.Must(template.ParseFiles("templates/graph.html"))

type UsageLogger interface {
	AddUsage(usage Usage)
	Serialize() map[string]interface{}
}

func graphHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load timezone: %v", err), http.StatusInternalServerError)
	}

	day := BeginningOfDay(time.Now().In(loc))
	newestDay := day
	if r.FormValue("ts") != "" {
		i, err := strconv.ParseInt(r.FormValue("ts"), 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse ts: %v", err), http.StatusBadRequest)
		}
		day = BeginningOfDay(time.Unix(i, 0).In(loc))
	}

	q := datastore.NewQuery("HourlyUsage").
		Filter("At >", day).
		Filter("At <", day.Add(time.Hour*24)).
		Order("-At")
	var usages []HourlyUsage
	_, err = q.GetAll(c, &usages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query usage logs: %v", err), http.StatusInternalServerError)
		return
	}

	logger := MakeUsageLogger()
	timestampByHostname := make(map[string][]int64)
	for _, hourlyUsage := range usages {
		for _, usage := range hourlyUsage.Events {
			logger.AddUsage(usage)
			timestampByHostname[usage.Hostname] = append(timestampByHostname[usage.Hostname],
				usage.At.Unix())
		}
	}

	// Make sure graph starts and ends at midnight by adding a pseudo-interval at the
	// beginning and end.
	allIntervals := []map[string]interface{}{
		map[string]interface{}{
			"label": "spacers",
			"times": []map[string]int64{
				map[string]int64{
					"starting_time": day.Unix() * 1000,
					"ending_time":   day.Unix()*1000 + 1,
				},
				map[string]int64{
					"starting_time": day.Add(time.Hour*24).Unix()*1000 - 1,
					"ending_time":   day.Add(time.Hour*24).Unix() * 1000,
				},
			},
		},
	}
	var total time.Duration
	for hostname, timestamps := range timestampByHostname {
		sort.Sort(int64Slice(timestamps))
		var intervals []map[string]int64
		last_ts := timestamps[0]
		interval_start := last_ts
		for _, timestamp := range timestamps {
			if timestamp > last_ts+int64(LogInterval.Seconds())+2 {
				intervals = append(intervals, map[string]int64{
					"starting_time": interval_start * 1000,
					"ending_time":   last_ts * 1000,
				})
				total += time.Duration(last_ts-interval_start) * time.Second
				interval_start = timestamp
			}
			last_ts = timestamp
		}
		intervals = append(intervals, map[string]int64{
			"starting_time": interval_start * 1000,
			"ending_time":   last_ts * 1000,
		})
		total += time.Duration(last_ts-interval_start) * time.Second
		allIntervals = append(allIntervals, map[string]interface{}{
			"label": hostname,
			"times": intervals,
		})
	}

	data := make(map[string]interface{})
	data["Usage"] = logger.Serialize()
	data["Intervals"] = allIntervals
	data["Total"] = total
	data["Date"] = day.Format("2006-01-02")
	data["Timestamp"] = day.Unix()
	data["NewestTimestamp"] = newestDay.Unix()
	data["OldestTimestamp"] = 1396738800

	if err := graphTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func BeginningOfHour(t time.Time) time.Time {
	return t.Truncate(time.Hour)
}

func BeginningOfDay(t time.Time) time.Time {
	d := time.Duration(-t.Hour()) * time.Hour
	return BeginningOfHour(t).Add(d)
}

func MakeUsageLogger() UsageLogger {
	l := usageLogger{}
	l.apps = make(map[string]UsageLogger)
	l.apps["chrome"] = MakeChromeUsage()
	l.apps["sublime_text"] = MakeSublimeUsage()
	return &l
}

type usageLogger struct {
	apps map[string]UsageLogger
}

func (l *usageLogger) AddUsage(usage Usage) {
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

func MakeAppUsage(process string) UsageLogger {
	return &appUsage{
		process:    process,
		categories: make(map[string]time.Duration),
	}
}

type appUsage struct {
	process    string
	categories map[string]time.Duration
}

func (a *appUsage) AddUsage(usage Usage) {
	a.categories[usage.Focused.WindowTitle] += LogInterval
}

func (a *appUsage) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"name":     a.process,
		"children": serializeChildren(a.categories),
	}
}

func MakeChromeUsage() UsageLogger {
	return &chromeUsage{
		sites: make(map[string]time.Duration),
	}
}

type chromeUsage struct {
	sites map[string]time.Duration
}

func (c *chromeUsage) AddUsage(usage Usage) {
	host := tryGetHostname(usage.Focused.WindowTitle)
	c.sites[host] += LogInterval
}

func (c *chromeUsage) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"name":     "chrome",
		"children": serializeChildren(c.sites),
	}
}

func MakeSublimeUsage() UsageLogger {
	return &sublimeUsage{
		projects: make(map[string]time.Duration),
	}
}

type sublimeUsage struct {
	projects map[string]time.Duration
}

func (s *sublimeUsage) AddUsage(usage Usage) {
	title := usage.Focused.WindowTitle
	project := "misc"
	if strings.Contains(title, "~/Dropbox/Programmieren") || strings.Contains(title,
		"~/Programmieren") {
		project = strings.Replace(title, "~/Dropbox/Programmieren/", "", -1)
		project = strings.Replace(project, "~/Programmieren/", "", -1)
		project = strings.Split(project, "/")[0]
	}
	s.projects[project] += LogInterval
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
	return strings.Replace(strings.Replace(host, "www", "", -1), ".com", "", -1)
}

type int64Slice []int64

func (a int64Slice) Len() int           { return len(a) }
func (a int64Slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a int64Slice) Less(i, j int) bool { return a[i] < a[j] }
