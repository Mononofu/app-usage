package usage

import (
	"fmt"
	"net/http"
)

func init() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/log/", logHandler)
	http.HandleFunc("/list/", listHandler)
	http.HandleFunc("/midi/", midiHandler)
	http.HandleFunc("/graph/", graphHandler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}
