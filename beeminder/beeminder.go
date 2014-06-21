package beeminder

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	auth_token = "Gh9NmVNeZzjJf6pz5cxS"
	username   = "mononofu"
)

func Update(goal string, value float64) error {
	url := fmt.Sprintf("https://www.beeminder.com/api/v1/users/%s/goals/%s/datapoints.json?value=%f&auth_token=%s",
		username, goal, value, auth_token)
	res, err := http.Post(url, "application/json`", nil)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("Beeminder request failed with status %d: %v", res.StatusCode, string(body))
	}
	return nil
}
