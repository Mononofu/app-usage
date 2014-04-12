package usage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
)

type RawUsage struct {
	Time         float64
	Focused      []RawApp
	Visible      []RawApp
	LastActivity float64 `json:"last_activity_ms"`
	Hostname     string
}

type RawApp struct {
	Name string
	Exec string
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var usages []RawUsage
	err := decoder.Decode(&usages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal request: %v", err), http.StatusBadRequest)
		return
	}

	usageByDay := make(map[time.Time][]Usage)
	for _, usage := range usages {
		if len(usage.Focused) == 0 {
			continue
		}

		focused := App{
			WindowTitle: usage.Focused[0].Name,
			Process:     usage.Focused[0].Exec,
		}
		var open []App
		for _, app := range usage.Visible {
			open = append(open, App{
				WindowTitle: app.Name,
				Process:     app.Exec,
			})
		}

		hostname := usage.Hostname
		if hostname == "" {
			hostname = "mononofu-laptop"
		}

		usageEvent := Usage{
			At:           time.Unix(int64(usage.Time), 0),
			Focused:      focused,
			LastActivity: time.Duration(int64(usage.LastActivity)) * time.Millisecond,
			Hostname:     hostname,
		}

		hour := usageEvent.At.Truncate(time.Hour)
		usageByDay[hour] = append(usageByDay[hour], usageEvent)
	}

	c := appengine.NewContext(r)
	for hour, usage := range usageByDay {
		key := datastore.NewKey(c, "HourlyUsage", "", hour.Unix(), nil)
		storedUsage := HourlyUsage{}
		err := datastore.Get(c, key, &storedUsage)
		if err != nil {
			storedUsage = HourlyUsage{
				At: hour,
			}
		}

		uniqueEvents := make(map[string]Usage)
		storedUsage.Events = append(storedUsage.Events, usage...)
		for _, event := range storedUsage.Events {
			uniqueEvents[fmt.Sprintf("%s__%s", event.Hostname, event.At)] = event
		}
		storedUsage.Events = []Usage{}
		for _, event := range uniqueEvents {
			storedUsage.Events = append(storedUsage.Events, event)
		}

		_, err = datastore.Put(c, key, &storedUsage)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save usage for %s: %v", hour, err),
				http.StatusInternalServerError)
			return
		}
	}
}
