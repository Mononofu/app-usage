package usage

import (
	"fmt"
	"net/http"

	"appengine"
	"appengine/datastore"

	"models"
)

func listHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q := datastore.NewQuery("HourlyUsage").Order("-At")
	var usages []models.HourlyUsage
	_, err := q.GetAll(c, &usages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query usage logs: %v", err), http.StatusInternalServerError)
	}

	fmt.Fprintf(w, "%d results: <br>\n<br>\n", len(usages))
	for _, usage := range usages {
		fmt.Fprintf(w, "%s: %d events<br>\n<br>\n", usage.At, len(usage.Events))
	}

	q = datastore.NewQuery("Piece").Project("Start", "Length").Order("-Start")
	var pieces []models.Piece
	_, err = q.GetAll(c, &pieces)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query midi logs: %v", err), http.StatusInternalServerError)
	}

	fmt.Fprintf(w, "\n\n\n%d results: <br>\n<br>\n", len(pieces))
	for _, piece := range pieces {
		fmt.Fprintf(w, "%s: played for %s<br>\n<br>\n", piece.Start, piece.Length)
	}
}
