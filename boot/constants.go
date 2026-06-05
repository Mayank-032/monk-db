package boot

// Manifest File based constants
const (
	MANIFESTFILEPATH = "./internal/sstable"
	MANIFESTFILENAME = "manifest.txt"
)

// Records directory based constants
const (
	RECORDSDIRPATH = "./internal/sstable/records"
)

// storage based constants
const (
	MEMTABLESIZE = 2000
)

// wal file based constants
const (
	WALFILENAME = "wal.db"
	WALFILEPATH = "./internal/storage"
)
