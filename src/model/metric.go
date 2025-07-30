package model

import (
	"time"
)

type MetricChanInfo struct {
	//Hash        types.Hash
	//View        types.View
	//ProposeTime *time.Time
	//CommitTime  *time.Time
	//
	//TraceId     string
	PayloadSize int
	Duration    time.Duration
}
