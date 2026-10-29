package models

import "time"

type Incident struct {
	Severity    string
	Status      string
	Description string
	Initiator   string
	SourceIP    string
	EventTime   *time.Time
	Raw         string
}
