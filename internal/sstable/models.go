package sstable

type Record struct {
	Key       string
	Value     string
	IsDeleted bool
}

type Pair struct {
	record  Record
	listIdx int
	idx     int
}
