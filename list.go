package usage

import (
	"fmt"
	"net/http"
	"sort"

	"appengine"
	"appengine/datastore"

	"common"
	"models"
)

func listHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q := datastore.NewQuery("HourlyUsage").Order("-At").Limit(2)
	var usages []models.HourlyUsage
	_, err := q.GetAll(c, &usages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query usage logs: %v", err), http.StatusInternalServerError)
		return
	}

	for _, usage := range usages {
		fmt.Fprintf(w, "\n\n\n%s", usage.At)
		sort.Sort(common.ByAt(usage.Events))
		for _, event := range usage.Events {
			fmt.Fprintf(w, "%s\n", event)
		}
	}
}
