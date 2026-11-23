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


// cursorsBucket holds, per peer ID, the last sync cursor received from it.
var cursorsBucket = []byte("cursors")


// peersBucket holds the multiaddr of every peer this node has connected to.
var peersBucket = []byte("peers")


// infoBucket holds facts about the store itself, such as its log id.
var infoBucket = []byte("info")


// logIDKey is the key of the log id in infoBucket.
var logIDKey = []byte("logId")


// buckets is every bucket New creates.
var buckets = [][]byte{
	eventsBucket,
	logBucket,
	metaBucket,
	tombstonesBucket,
	cursorsBucket,
	peersBucket,
	infoBucket,
}
