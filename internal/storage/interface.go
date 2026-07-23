package storage

import (
	"monk-db/internal/models"
)

type IDiskStorage interface {
	Flush(sstableRecords []models.Record) error
	Read(key string) (string, error)
	CountFiles() (int, error)
	Optimize() error
}
