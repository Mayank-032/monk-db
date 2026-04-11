package sstable

type Record struct {
	Key       string
	Value     string
	IsDeleted bool
}
