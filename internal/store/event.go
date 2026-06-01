package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type SystemEventsStore struct {
	db *DB
}

func NewSystemEventsStore(db *DB) *SystemEventsStore {
	return &SystemEventsStore{db: db}
}

func (s *SystemEventsStore) Create(event *model.SystemEvent) error {
	event.ID = uuid.New().String()
	event.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		`INSERT INTO system_events (id, event_type, severity, message, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		event.ID, event.EventType, string(event.Severity), event.Message, event.Metadata, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert system event: %w", err)
	}
	return nil
}

func (s *SystemEventsStore) List(limit, offset int, eventType, severity, dateFrom, dateTo string) ([]model.SystemEvent, int, error) {
	var conditions []string
	var args []interface{}

	if eventType != "" {
		conditions = append(conditions, "event_type = ?")
		args = append(args, eventType)
	}
	if severity != "" {
		conditions = append(conditions, "severity = ?")
		args = append(args, severity)
	}
	if dateFrom != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, dateTo)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM system_events %s", whereClause)
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count system events: %w", err)
	}

	selectQuery := fmt.Sprintf(
		"SELECT id, event_type, severity, message, metadata, created_at FROM system_events %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		whereClause,
	)
	queryArgs := append(args, limit, offset)

	rows, err := s.db.Query(selectQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list system events: %w", err)
	}
	defer rows.Close()

	var events []model.SystemEvent
	for rows.Next() {
		var e model.SystemEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Severity, &e.Message, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan system event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}

	return events, total, nil
}

func (s *SystemEventsStore) EventCounts() (map[string]int, error) {
	rows, err := s.db.Query("SELECT event_type, COUNT(*) FROM system_events GROUP BY event_type")
	if err != nil {
		return nil, fmt.Errorf("event counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		counts[eventType] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return counts, nil
}

func (s *SystemEventsStore) PurgeOlderThan(age time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-age).Format(time.RFC3339)
	result, err := s.db.Exec("DELETE FROM system_events WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("purge system events: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}
