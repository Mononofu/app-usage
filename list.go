package usage

import (
	"fmt"
	"net/http"

	"appengine"
	"appengine/datastore"
)

func listHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("HourlyUsage").Order("-At")

	var usages []HourlyUsage
	_, err := q.GetAll(c, &usages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query usage logs: %v", err), http.StatusInternalServerError)
	}

	fmt.Fprintf(w, "%d results: \n\n", len(usages))
	for _, usage := range usages {
		fmt.Fprintf(w, "%s: %d events\n\n", usage.At, len(usage.Events))
	}
}
