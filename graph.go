package usage

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"time"

	"appengine"
	"appengine/datastore"

	"common"
	"models"
	"usage"
)

var graphTemplate = template.Must(template.ParseFiles("templates/graph.html"))

func graphHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load timezone: %v", err), http.StatusInternalServerError)
		return
	}

	day := BeginningOfDay(time.Now().In(loc))
	newestDay := day
	if r.FormValue("ts") != "" {
		i, err := strconv.ParseInt(r.FormValue("ts"), 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse ts: %v", err), http.StatusBadRequest)
			return
		}
		day = BeginningOfDay(time.Unix(i, 0).In(loc))
	}

	q := datastore.NewQuery("HourlyUsage").
		Filter("At >=", day).
		Filter("At <", day.Add(time.Hour*24)).
		Order("At")
	var usages []models.HourlyUsage
	_, err = q.GetAll(c, &usages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query usage logs: %v", err), http.StatusInternalServerError)
		return
	}

	q = datastore.NewQuery("Piece").
		Filter("Start >=", day).
		Filter("Start <", day.Add(time.Hour*24)).
		Order("Start")
	var pieces []models.Piece
	_, err = q.GetAll(c, &pieces)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query midi logs: %v", err), http.StatusInternalServerError)
		return
	}

	logger, timestampByHostname, err := filterIdles(usages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to filter usage: %v", err), http.StatusInternalServerError)
		return
	}
	allIntervals, total := calculateIntervals(day, timestampByHostname)

	// Make the logger believe midi notes are app usage.
	var pianoIntervals []map[string]int64
	for _, piece := range pieces {
		curPos := piece.Start
		end := curPos.Add(piece.Length)
		for curPos.Before(end) {
			logger.AddUsage(models.Usage{Focused: models.App{Process: "piano"}})
			curPos = curPos.Add(common.LogInterval)
		}
		pianoIntervals = append(pianoIntervals, map[string]int64{
			"starting_time": piece.Start.Unix() * 1000,
			"ending_time":   end.Unix() * 1000,
		})
	}
	if pianoIntervals != nil {
		allIntervals = append(allIntervals, map[string]interface{}{
			"label": "piano",
			"times": pianoIntervals,
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

func filterIdles(usages []models.HourlyUsage) (usage.Logger, map[string][]int64, error) {
	logger, err := usage.MakeLogger()
	if err != nil {
		return nil, nil, err
	}

	timestampByHostname := make(map[string][]int64)
	var events []models.Usage
	// Simulating a deque by keeping an index to which elements from the beginning we've already
	// processed.
	i := 0
	for _, hourlyUsage := range usages {
		sort.Sort(byAt(hourlyUsage.Events))

		for _, usage := range hourlyUsage.Events {
			events = append(events, usage)

			if usage.LastActivity < common.IdleTimeout {
				for usage.At.Sub(events[i].At) > common.IdleTimeout {
					logger.AddUsage(events[i])
					timestampByHostname[events[i].Hostname] = append(timestampByHostname[events[i].Hostname],
						events[i].At.Unix())
					i++
				}
			} else {
				events = []models.Usage{}
				i = 0
			}
		}
	}
	for _, event := range events {
		logger.AddUsage(event)
		timestampByHostname[event.Hostname] = append(timestampByHostname[event.Hostname],
			event.At.Unix())
	}

	return logger, timestampByHostname, nil
}

func calculateIntervals(day time.Time, timestampByHostname map[string][]int64) ([]map[string]interface{}, time.Duration) {
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
			if timestamp > last_ts+int64(common.LogInterval.Seconds())+2 {
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

	return allIntervals, total
}

func BeginningOfHour(t time.Time) time.Time {
	return t.Truncate(time.Hour)
}

func BeginningOfDay(t time.Time) time.Time {
	d := time.Duration(-t.Hour()) * time.Hour
	return BeginningOfHour(t).Add(d)
}

type int64Slice []int64

func (a int64Slice) Len() int           { return len(a) }
func (a int64Slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a int64Slice) Less(i, j int) bool { return a[i] < a[j] }

type byAt []models.Usage

func (a byAt) Len() int           { return len(a) }
func (a byAt) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAt) Less(i, j int) bool { return a[i].At.Before(a[j].At) }
