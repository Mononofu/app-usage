package usage

import (
	"net/http"
)

func init() {
	http.HandleFunc("/", graphHandler)
	http.HandleFunc("/log/", logHandler)
	http.HandleFunc("/list/", listHandler)
	http.HandleFunc("/midi/", midiHandler)
}
