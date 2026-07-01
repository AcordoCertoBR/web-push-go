package webpush

type MessageActions struct {
	Action string `json:"action,omitempty"`
	Title  string `json:"title,omitempty"`
	Icon   string `json:"icon,omitempty"`
}

type MessageDir = string

const (
	DirAuto MessageDir = "auto"
	DirLTR  MessageDir = "ltr"
	DirRTL  MessageDir = "rtl"
)

type MessageOptions struct {
	Actions            []MessageActions `json:"actions,omitempty"`
	Badge              string           `json:"badge,omitempty"`
	Body               string           `json:"body,omitempty"`
	Data               any              `json:"data,omitempty"`
	Dir                MessageDir       `json:"dir,omitempty"`
	Icon               string           `json:"icon,omitempty"`
	Image              string           `json:"image,omitempty"`
	Lang               string           `json:"lang,omitempty"`
	Renotify           bool             `json:"renotify,omitempty"`
	RequireInteraction bool             `json:"requireInteraction,omitempty"`
	Silent             bool             `json:"silent,omitempty"`
	Tag                string           `json:"tag,omitempty"`
	Timestamp          int64            `json:"timestamp,omitempty"`
	// Vibrate is a vibration pattern in milliseconds (alternating
	// vibration/pause), per the Notification API — not an on/off flag.
	Vibrate []int `json:"vibrate,omitempty"`
}

// Check for more details here: https://developer.mozilla.org/en-US/docs/Web/API/ServiceWorkerRegistration/showNotification
type Message struct {
	Title   string         `json:"title,omitempty"`
	Options MessageOptions `json:"options"`
}
