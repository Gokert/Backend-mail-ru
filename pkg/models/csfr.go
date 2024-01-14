package models

import "time"

type Csrf struct {
	SID       string
	ExpiresAt time.Time
}
