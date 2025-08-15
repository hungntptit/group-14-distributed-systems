package model

import "time"

type PeerInfo struct {
	URL      string
	LastSeen time.Time
}
