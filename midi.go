package usage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"appengine"
	"appengine/datastore"

	"models"
)

type RawNote struct {
	Status            int16
	Data1             int16
	Data2             int16
	Data3             int16
	RelativeTimestamp int32   `json:"relative_timestamp"`
	AbsoluteTimestamp float64 `json:"absolute_timestamp"`
}

func midiHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var notes []RawNote
	err := decoder.Decode(&notes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal request: %v", err), http.StatusBadRequest)
		return
	}

	if len(notes) == 0 {
		http.Error(w, "At least one note required.", http.StatusBadRequest)
		return
	}

	piece := models.Piece{}
	for _, note := range notes {
		piece.Notes = append(piece.Notes, models.Note{
			Status:            note.Status,
			Data1:             note.Data1,
			Data2:             note.Data2,
			Data3:             note.Data3,
			RelativeTimestamp: note.RelativeTimestamp,
			AbsoluteTimestamp: time.Unix(int64(note.AbsoluteTimestamp),
				int64(note.AbsoluteTimestamp*1e9)%1e9),
		})
	}

	sort.Sort(byAbsoluteTime(piece.Notes))
	piece.Start = piece.Notes[0].AbsoluteTimestamp
	piece.Length = piece.Notes[len(piece.Notes)-1].AbsoluteTimestamp.Sub(piece.Start)

	c := appengine.NewContext(r)
	key := datastore.NewKey(c, "Piece", "", piece.Start.Unix(), nil)
	_, err = datastore.Put(c, key, &piece)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save piece for %s: %v", piece.Start, err), http.StatusInternalServerError)
		c.Errorf("Failed to save piece for %s: %v", piece.Start, err)
		return
	}

	if err := beeminder.update("piano", piece.Length.Minutes()); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update beeminder: %v", err), http.StatusInternalServerError)
		c.Errorf("Failed to update beeminder: %v", err)
		return
	}
}

type byAbsoluteTime []models.Note

func (a byAbsoluteTime) Len() int      { return len(a) }
func (a byAbsoluteTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byAbsoluteTime) Less(i, j int) bool {
	return a[i].AbsoluteTimestamp.Before(a[j].AbsoluteTimestamp)
}
