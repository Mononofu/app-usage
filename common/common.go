package common

import (
	"time"

	"models"
)

const LogInterval = 10 * time.Second
const IdleTimeout = 5 * time.Minute

type ByAt []models.Usage

func (a ByAt) Len() int           { return len(a) }
func (a ByAt) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAt) Less(i, j int) bool { return a[i].At.Before(a[j].At) }
