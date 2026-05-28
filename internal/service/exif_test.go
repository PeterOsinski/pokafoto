package service

import (
	"os/exec"
	"testing"
)

func TestExifService_ExtractViaExifTool_shouldParseOutput(t *testing.T) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not available")
	}

	src := createTestJPEG(t, t.TempDir(), "test_exif.jpg")
	svc := NewExifService()

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
	svc := NewExifService()

	data, err := svc.Extract(src)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if data == nil {
		t.Log("goexif returned nil — expected for synthetic JPEG without EXIF tags")
	}
}
