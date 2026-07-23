package constants

// wal file based constants
const (
	WAL_FILENAME = "wal.db"
	WAL_FILEPATH = "/internal/storage"
)

// Records directory based constants
const (
	RECORDS_DIRPATH = "./internal/sstable/records"
	LEVELZERO_PATH  = "/l0"
)

// Manifest File based constants
const (
	MANIFEST_FILEPATH = "./internal/sstable"
	MANIFEST_FILENAME = "manifest.txt"
)
