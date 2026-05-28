package store

import (
	"fmt"
)

type GeoStore struct {
	db *DB
}

func NewGeoStore(db *DB) *GeoStore {
	return &GeoStore{db: db}
}

type GeoPoint struct {
	FileID       string  `json:"fileId"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	ThumbnailURL string  `json:"thumbnailUrl"`
	TakenAt      string  `json:"takenAt"`
}

type GeoBounds struct {
	LatMin float64
	LatMax float64
	LonMin float64
	LonMax float64
}

func (s *GeoStore) GetPoints(userID string, bounds GeoBounds) ([]GeoPoint, error) {
	rows, err := s.db.Query(`
		SELECT f.id, e.gps_latitude, e.gps_longitude, f.taken_at
		FROM files f
		INNER JOIN exif e ON f.id = e.file_id
		WHERE f.user_id = ?
		  AND f.is_deleted = 0
		  AND e.gps_latitude BETWEEN ? AND ?
		  AND e.gps_longitude BETWEEN ? AND ?
		ORDER BY f.taken_at DESC
	`, userID, bounds.LatMin, bounds.LatMax, bounds.LonMin, bounds.LonMax)
	if err != nil {
		return nil, fmt.Errorf("query geo points: %w", err)
	}
	defer rows.Close()

	var points []GeoPoint
	for rows.Next() {
		var p GeoPoint
		var takenAt *string
		if err := rows.Scan(&p.FileID, &p.Latitude, &p.Longitude, &takenAt); err != nil {
			return nil, fmt.Errorf("scan geo point: %w", err)
		}
		p.ThumbnailURL = fmt.Sprintf("/api/v1/thumb/%s/sm.jpg", p.FileID)
		if takenAt != nil {
			p.TakenAt = *takenAt
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
