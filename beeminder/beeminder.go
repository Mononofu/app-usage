package beeminder

import (
	"net/http"
)

const (
	auth_token = "Gh9NmVNeZzjJf6pz5cxS"
	username   = "mononofu"
)

func update(goal string, value float64) error {
	url := fmt.Sprintf("https://www.beeminder.com/api/v1/users/%s/goals/%s/datapoints.json?value=%s",
		username, goal, value)
	_, err := http.Post(url, "application/json`", nil)
	return err
}
