package flag

import "time"

const (
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
	StatusSkipped  = "skipped"
	StatusQueued   = "queued"
)

type Flag struct {
	Flag                string    `json:"flag"`
	Exploit             string    `json:"sploit"`
	Team                string    `json:"team"`
	ReceivedTime        time.Time `json:"receivedTime"`
	SentTime            time.Time `json:"sentTime,omitempty"`
	Status              string    `json:"status"`
	CheckSystemResponse string    `json:"checkSystemResponse"`
}
