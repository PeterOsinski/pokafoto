package store

import (
	"database/sql"
	"fmt"

	"github.com/drive/drive/internal/model"
)

type ExifStore struct {
	db *DB
}

func NewExifStore(db *DB) *ExifStore {
	return &ExifStore{db: db}
}

func (s *ExifStore) Create(exif *model.ExifData) error {
	_, err := s.db.Exec(
		`INSERT INTO exif (file_id, camera_make, camera_model, lens_make, lens_model, focal_length, aperture, shutter_speed, iso, date_taken, gps_latitude, gps_longitude, gps_altitude, orientation, color_space, flash, software, raw_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		exif.FileID, exif.CameraMake, exif.CameraModel, exif.LensMake, exif.LensModel, exif.FocalLength, exif.Aperture, exif.ShutterSpeed, exif.ISO, exif.DateTaken, exif.GPSLatitude, exif.GPSLongitude, exif.GPSAltitude, exif.Orientation, exif.ColorSpace, exif.Flash, exif.Software, exif.RawJSON,
	)
	if err != nil {
		return fmt.Errorf("insert exif: %w", err)
	}
	return nil
}

func (s *ExifStore) FindByFileID(fileID string) (*model.ExifData, error) {
	e := &model.ExifData{}

	err := s.db.QueryRow(
		`SELECT file_id, camera_make, camera_model, lens_make, lens_model, focal_length, aperture, shutter_speed, iso, date_taken, gps_latitude, gps_longitude, gps_altitude, orientation, color_space, flash, software, raw_json FROM exif WHERE file_id = ?`,
		fileID,
	).Scan(&e.FileID, &e.CameraMake, &e.CameraModel, &e.LensMake, &e.LensModel, &e.FocalLength, &e.Aperture, &e.ShutterSpeed, &e.ISO, &e.DateTaken, &e.GPSLatitude, &e.GPSLongitude, &e.GPSAltitude, &e.Orientation, &e.ColorSpace, &e.Flash, &e.Software, &e.RawJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find exif by file id: %w", err)
	}

	return e, nil
}
