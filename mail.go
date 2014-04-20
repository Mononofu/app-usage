package usage

import (
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"

	"models"
)

func init() {
	http.HandleFunc("/_ah/mail/", incomingMail)
}

func incomingMail(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	defer r.Body.Close()
	msg, err := mail.ReadMessage(r.Body)
	if err != nil {
		c.Errorf("Error reading body: %v", err)
		return
	}

	if len(msg.Header["Content-Type"]) != 1 {
		c.Errorf("Expected content type exactly once", http.StatusBadRequest)
		return
	}

	mediatype, params, err := mime.ParseMediaType(msg.Header["Content-Type"][0])
	if err != nil {
		c.Errorf("Error parsing content type: %v", err)
		return
	}

	if !strings.Contains(mediatype, "multipart") {
		c.Errorf("Expected mail with attachment")
		return
	}

	var csvs []Part
	reader := multipart.NewReader(msg.Body, params["boundary"])
	for nestedPart, err := reader.NextPart(); err == nil; nestedPart, err = reader.NextPart() {
		childParts, err := flatten(nestedPart)
		if err != nil {
			c.Errorf("Failed to parse parts: %v", err)
			return
		}
		for _, part := range childParts {
			if strings.Contains(part.Header.Get("Content-Type"), "text/csv") {
				csvs = append(csvs, part)
			}
		}
	}

	if len(csvs) == 0 {
		c.Errorf("Expected at least one CSV")
		return
	}

	for _, file := range csvs {
		content := file.Content
		if file.Header.Get("Content-Transfer-Encoding") == "base64" {
			data, err := base64.StdEncoding.DecodeString(content)
			if err != nil {
				c.Errorf("Failed to decode base64: %v", err)
				continue
			}
			content = string(data)
		} else {
			content = strings.Trim(content, " \n\r")
		}

		reader := csv.NewReader(strings.NewReader(content))
		records, err := reader.ReadAll()
		if err != nil {
			c.Errorf("Failed to read CSV: %v", err)
			continue
		}

		rows := csvMap(records)

		var keys []*datastore.Key
		var tubes []*models.Tube
		for _, row := range rows {
			journey := row["Journey/Action"]
			journey = strings.Replace(journey, "[London Underground]", "", -1)
			fromTo := strings.Split(journey, " to ")

			dateFormat := "02-Jan-2006-15:04"

			start, err := time.Parse(dateFormat, row["Date"]+"-"+row["Start Time"])
			if err != nil {
				c.Errorf("Failed to parse time: %v", err)
				continue
			}
			end, err := time.Parse(dateFormat, row["Date"]+"-"+row["End Time"])
			if err != nil {
				c.Errorf("Failed to parse time: %v", err)
				continue
			}

			tube := models.Tube{
				From:  fromTo[0],
				To:    fromTo[1],
				Start: start,
				End:   end,
			}
			tubes = append(tubes, &tube)
			keys = append(keys, datastore.NewKey(c, "Tube", "", tube.Start.Unix(), nil))
		}

		_, err = datastore.PutMulti(c, keys, tubes)
		if err != nil {
			c.Errorf("Failed to save tube journey: %v", err)
			continue
		}
	}
}

func csvMap(records [][]string) []map[string]string {
	var rows []map[string]string
	header := records[0]
	for _, row := range records[1:] {
		values := make(map[string]string)
		for i := range header {
			values[header[i]] = row[i]
		}
		rows = append(rows, values)
	}

	return rows
}

func flatten(part *multipart.Part) ([]Part, error) {
	var parts []Part

	mediatype, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("Error parsing content type: %v", err)
	}

	if strings.Contains(mediatype, "multipart") {
		reader := multipart.NewReader(part, params["boundary"])
		for part, err := reader.NextPart(); err == nil; part, err = reader.NextPart() {
			childParts, err := flatten(part)
			if err != nil {
				return nil, err
			}
			parts = append(parts, childParts...)
		}
	} else {
		b, err := ioutil.ReadAll(part)
		if err != nil {
			return nil, err
		}
		parts = append(parts, Part{
			Content:  string(b),
			Filename: part.FileName(),
			Header:   part.Header,
		})
	}
	return parts, nil
}

type Part struct {
	Content  string
	Filename string
	Header   textproto.MIMEHeader
}
