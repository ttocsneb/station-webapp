package station

import "time"

type sensorValue struct {
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

type weatherMessage struct {
	Time    time.Time                `json:"time"`
	ID      string                   `json:"id"`
	Sensors map[string][]sensorValue `json:"sensors"`
}

type requestMessage struct {
	Action string `json:"action"`
}
