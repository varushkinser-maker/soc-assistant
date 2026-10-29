package models

type Authentication struct {
	SubjectName string `json:"subject_name"`
}
type Request_metadata struct {
	RemoteAddress string `json:"remote_address"`
}
type Extevent struct {
	Id              string           `json:"event_id"`
	EventName       string           `json:"event_name"`
	EventType       string           `json:"event_type"`
	EventTime       string           `json:"event_time"`
	Authenication   Authentication   `json:"authentication"`
	RequestMetadata Request_metadata `json:"request_metadata"`
	Raw             string           `json:"-"`
}

func (e *Extevent) TypeName() string {
	if e.EventType == "" {
		return e.EventName
	}
	return e.EventType
}
