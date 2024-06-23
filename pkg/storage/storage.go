package storage

import (
	"time"

	"github.com/VaiTon/Agamennone/pkg/flag"
)

type FlagStorage interface {
	Init() error
	GetLastFlags(limit int) ([]flag.Flag, error)
	GetByStatus(status string, limit int) ([]flag.Flag, error)
	GetStatisticsV1() (StatisticsV1, error)
	UpdateSentFlag(flag, status, checkSystemResponse string, sentTime time.Time) error
	InsertFlags(flags []flag.Flag) (int64, error)
	GetStatsByExploit() ([]StatsByExploit, error)
}

type StatsByExploit struct {
	Hour    time.Time
	Exploit string
	Status  string
	Count   int
}

type StatisticsV1 struct {
	TotalFlags         int
	TotalFlagsByStatus map[string]int
}
