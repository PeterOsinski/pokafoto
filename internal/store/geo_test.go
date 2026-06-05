package store

import (
	"fmt"
	"testing"

	"github.com/drive/drive/internal/model"
)

func createTestFileWithGPS(t *testing.T, db *DB, userID string, lat, lon float64) *model.File {
	t.Helper()
	fs := NewFileStore(db)
	es := NewExifStore(db)

	f := &model.File{
		UserID:       userID,
		Filename:     "2024/07/geo.jpg",
		OriginalName: "geo.jpg",
		Path:         "2024/07",
		SizeBytes:    2048,
		MimeType:     "image/jpeg",
		SHA256:       makeSHA256(fmt.Sprintf("%f,%f", lat, lon)),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(f)

	e := &model.ExifData{
		FileID:       f.ID,
		GPSLatitude:  &lat,
		GPSLongitude: &lon,
	}
	es.Create(e)

	return f
}

func TestGeoStore_GetPoints_shouldReturnPointsInBBox(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	gs := NewGeoStore(db)

	user := createTestUser(t, us)
	createTestFileWithGPS(t, db, user.ID, 52.2297, 21.0122)

	points, err := gs.GetPoints(user.ID, GeoBounds{
		LatMin: 50,
		LatMax: 55,
		LonMin: 20,
		LonMax: 25,
	})
	if err != nil {
		t.Fatalf("get points: %v", err)
	}
	if len(points) == 0 {
		t.Error("expected points in bbox")
	}
	if len(points) > 0 && points[0].ThumbnailURL == "" {
		t.Error("expected ThumbnailURL to be set")
	}
}

func TestGeoStore_GetPoints_shouldReturnEmptyOutsideBBox(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	gs := NewGeoStore(db)

	user := createTestUser(t, us)
	createTestFileWithGPS(t, db, user.ID, 52.0, 21.0)

	points, err := gs.GetPoints(user.ID, GeoBounds{
		LatMin: 0,
		LatMax: 1,
		LonMin: 0,
		LonMax: 1,
	})
	if err != nil {
		t.Fatalf("get points: %v", err)
	}
	if len(points) != 0 {
		t.Errorf("expected 0 points outside bbox, got %d", len(points))
	}
}

func TestGeoStore_GetPoints_shouldFilterByUser(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	gs := NewGeoStore(db)

	user1 := createTestUser(t, us)
	user2, _ := us.Create("geo_user_"+t.Name()+"_2", "password123", model.RoleMember, nil)
	createTestFileWithGPS(t, db, user1.ID, 52.0, 21.0)
	createTestFileWithGPS(t, db, user2.ID, 52.0, 21.0)

	points, err := gs.GetPoints(user1.ID, GeoBounds{
		LatMin: 50, LatMax: 55,
		LonMin: 20, LonMax: 25,
	})
	if err != nil {
		t.Fatalf("get points: %v", err)
	}
	if len(points) != 1 {
		t.Errorf("expected 1 point for user1, got %d", len(points))
	}
}

func TestGeoStore_GetPoints_shouldExcludeDeletedFiles(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	gs := NewGeoStore(db)

	user := createTestUser(t, us)
	createTestFileWithGPS(t, db, user.ID, 52.0, 21.0)

	deleted := createTestFileWithGPS(t, db, user.ID, 53.0, 22.0)
	db.Exec("UPDATE files SET is_deleted = 1 WHERE id = ?", deleted.ID)

	points, err := gs.GetPoints(user.ID, GeoBounds{
		LatMin: 50, LatMax: 55,
		LonMin: 20, LonMax: 25,
	})
	if err != nil {
		t.Fatalf("get points: %v", err)
	}
	if len(points) != 1 {
		t.Errorf("expected 1 point excluding deleted, got %d", len(points))
	}
}

func TestGeoStore_GetPoints_shouldReturnEmptyForUnknownUser(t *testing.T) {
	db := OpenTestDB(t)
	gs := NewGeoStore(db)

	points, err := gs.GetPoints("nonexistent", GeoBounds{
		LatMin: -90, LatMax: 90,
		LonMin: -180, LonMax: 180,
	})
	if err != nil {
		t.Fatalf("get points: %v", err)
	}
	if len(points) != 0 {
		t.Errorf("expected 0 points, got %d", len(points))
	}
}
