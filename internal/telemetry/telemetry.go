// Package telemetry holds the structured UI push payloads shared by the v1
// server loop and the v2 session layer (transfer progress + activity rows).
// Keeping them here avoids an import cycle between those two packages.
package telemetry

// Transfer states.
const (
	TransferActive    = "active"
	TransferDone      = "done"
	TransferError     = "error"
	TransferCancelled = "cancelled"
)

// Transfer directions.
const (
	DirIn  = "in"
	DirOut = "out"
)

// Transfer is one in-flight (or just finished) file movement.
type Transfer struct {
	ID     string `json:"id"`
	Dir    string `json:"dir"`
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	Total  int64  `json:"total"`
	BPS    int64  `json:"bps"`
	State  string `json:"state"`
	Device string `json:"device,omitempty"`
}

// Activity kinds (Console EVENT column).
const (
	KindServerStarted   = "server_started"
	KindServerStopped   = "server_stopped"
	KindPhoneConnected  = "phone_connected"
	KindPhoneDisconnect = "phone_disconnected"
	KindFileSent        = "file_sent"
	KindFileReceived    = "file_received"
	KindMatchesSent     = "matches_sent"
	KindError           = "error"
	KindImport          = "import"
)

// Activity is one structured row for the Console activity table.
type Activity struct {
	TS     int64  `json:"ts"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
	Status string `json:"status"`
	Device string `json:"device,omitempty"`
}

// Observer receives transfer + activity pushes.
type Observer interface {
	Transfer(Transfer)
	Activity(Activity)
}
