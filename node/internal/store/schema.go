package store


// --- GLOBALS ---

// eventsBucket holds every stored event, raw, keyed by its eventId.
var eventsBucket = []byte("events")


// logBucket maps the local sequence number to the eventId.
var logBucket = []byte("log")


// metaBucket maps an eventId to its sequence and local time of receipt.
var metaBucket = []byte("meta")


// tombstonesBucket maps a pruned eventId to its pruning time and author.
var tombstonesBucket = []byte("tombstones")


// buckets is every bucket New creates.
var buckets = [][]byte{
	eventsBucket,
	logBucket,
	metaBucket,
	tombstonesBucket,
}
