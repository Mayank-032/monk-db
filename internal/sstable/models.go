package sstable

import "monk-db/internal/models"

type Pair struct {
	record  models.Record
	listIdx int
	idx     int
}
