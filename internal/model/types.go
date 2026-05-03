package model

import "time"

type Event struct {
	Source    string
	Raw       string
	Timestamp time.Time
}

type Cluster struct {
	ID       uint64    `json:"id"`
	Pattern  string    `json:"pattern"`
	Count    uint64    `json:"count"`
	LastSeen time.Time `json:"last_seen"`
	Sample   string    `json:"sample"`
}

type PatternObservation struct {
	Event        Event
	Cluster      *Cluster
	WindowCount  int
	Baseline     float64
	IsNewPattern bool
	IsAnomaly    bool
	Reason       string
}

type Alert struct {
	Key      string
	Title    string
	Body     string
	Severity string
	Source   string
	Pattern  string
}
