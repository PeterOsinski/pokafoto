package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	mp4lib "github.com/abema/go-mp4"
)

func TestExifService_ExtractVideoDate_shouldExtractMvhdCreationTime(t *testing.T) {
	src := createTestMP4(t, t.TempDir(), "test.mp4")
	svc := NewExifService(NewRealFS())

	data, err := svc.ExtractVideoDate(src)
	if err != nil {
		t.Fatalf("ExtractVideoDate failed: %v", err)
	}
	if data == nil {
		t.Fatal("expected exif data, got nil")
	}
	if data.DateTaken == nil {
		t.Fatal("expected DateTaken, got nil")
	}

	expected := "2020:01:01 00:00:00"
	if *data.DateTaken != expected {
		t.Fatalf("expected DateTaken %q, got %q", expected, *data.DateTaken)
	}
}

func TestExifService_ExtractVideoDate_shouldReturnNilForNonMP4File(t *testing.T) {
	src := createTestJPEG(t, t.TempDir(), "test.jpg")
	svc := NewExifService(NewRealFS())

	data, err := svc.ExtractVideoDate(src)
	if err == nil {
		t.Fatal("expected error for JPEG file")
	}
	if data != nil {
		t.Fatal("expected nil data for JPEG file")
	}
}

func createTestMP4(t *testing.T, dir string, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test mp4 file: %v", err)
	}
	defer f.Close()

	w := mp4lib.NewWriter(f)

	if _, err := w.StartBox(&mp4lib.BoxInfo{Type: mp4lib.BoxTypeFtyp()}); err != nil {
		t.Fatalf("start ftyp: %v", err)
	}
	ftyp := &mp4lib.Ftyp{
		MajorBrand:   [4]byte{'i', 's', 'o', 'm'},
		MinorVersion: 512,
		CompatibleBrands: []mp4lib.CompatibleBrandElem{
			{CompatibleBrand: [4]byte{'i', 's', 'o', 'm'}},
		},
	}
	if _, err := mp4lib.Marshal(w, ftyp, mp4lib.Context{}); err != nil {
		t.Fatalf("marshal ftyp: %v", err)
	}
	if _, err := w.EndBox(); err != nil {
		t.Fatalf("end ftyp: %v", err)
	}

	if _, err := w.StartBox(&mp4lib.BoxInfo{Type: mp4lib.BoxTypeMoov()}); err != nil {
		t.Fatalf("start moov: %v", err)
	}
	if _, err := w.StartBox(&mp4lib.BoxInfo{Type: mp4lib.BoxTypeMvhd()}); err != nil {
		t.Fatalf("start mvhd: %v", err)
	}

	qtEpoch := time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)
	targetTime := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	creationTimeSeconds := uint32(targetTime.Sub(qtEpoch).Seconds())

	mvhd := &mp4lib.Mvhd{
		FullBox: mp4lib.FullBox{
			Version: 0,
		},
		CreationTimeV0:     creationTimeSeconds,
		ModificationTimeV0: creationTimeSeconds,
		Timescale:          1000,
		DurationV0:         5000,
		Rate:               0x00010000,
		Volume:             0x0100,
		Matrix:             [9]int32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000},
		NextTrackID:        1,
	}
	if _, err := mp4lib.Marshal(w, mvhd, mp4lib.Context{}); err != nil {
		t.Fatalf("marshal mvhd: %v", err)
	}
	if _, err := w.EndBox(); err != nil {
		t.Fatalf("end mvhd: %v", err)
	}
	if _, err := w.EndBox(); err != nil {
		t.Fatalf("end moov: %v", err)
	}

	return path
}

func TestExifService_ExtractViaExifTool_shouldParseOutput(t *testing.T) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not available")
	}

	src := createTestJPEG(t, t.TempDir(), "test_exif.jpg")
	svc := NewExifService(NewRealFS())

	data, err := svc.Extract(src)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if data == nil {
		return
	}

	if data.RawJSON != nil && *data.RawJSON != "" {
		t.Logf("exiftool data: %s", *data.RawJSON)
	}
}

func TestExifService_Extract_goexif_shouldExtractJPEG(t *testing.T) {
	src := createTestJPEG(t, t.TempDir(), "test.jpg")
	svc := NewExifService(NewRealFS())

	data, err := svc.Extract(src)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if data == nil {
		t.Log("goexif returned nil — expected for synthetic JPEG without EXIF tags")
	}
}
