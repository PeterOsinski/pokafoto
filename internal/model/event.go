package model

type EventSeverity string

const (
	SeverityInfo    EventSeverity = "info"
	SeverityWarning EventSeverity = "warning"
	SeverityError   EventSeverity = "error"
)

type SystemEvent struct {
	ID        string         `json:"id" db:"id"`
	EventType string         `json:"event_type" db:"event_type"`
	Severity  EventSeverity  `json:"severity" db:"severity"`
	Message   string         `json:"message" db:"message"`
	Metadata  *string        `json:"metadata,omitempty" db:"metadata"`
	CreatedAt string         `json:"created_at" db:"created_at"`
}
