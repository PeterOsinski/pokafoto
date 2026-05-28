package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"

	"github.com/drive/drive/internal/model"
	"github.com/rwcarlsen/goexif/exif"
)

type ExifService struct{}

func NewExifService() *ExifService {
	return &ExifService{}
}

type exifToolEntry struct {
	Make            string `json:"Make"`
	Model           string `json:"Model"`
	LensModel       string `json:"LensModel"`
	LensMake        string `json:"LensMake"`
	FocalLength     string `json:"FocalLength"`
	FNumber         float64 `json:"FNumber"`
	ExposureTime    string `json:"ExposureTime"`
	ISO             int    `json:"ISO"`
	DateTimeOriginal string `json:"DateTimeOriginal"`
	GPSLatitude     float64 `json:"GPSLatitude"`
	GPSLongitude    float64 `json:"GPSLongitude"`
	GPSAltitude     float64 `json:"GPSAltitude"`
	Orientation     int    `json:"Orientation"`
	ColorSpace      string `json:"ColorSpace"`
	Flash           int    `json:"Flash"`
	Software        string `json:"Software"`
}

func (s *ExifService) Extract(filePath string) (*model.ExifData, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file for exif: %w", err)
	}
	defer f.Close()

	var data model.ExifData

	x, err := exif.Decode(f)
	if err != nil {
		data, fallbackErr := s.extractViaExifTool(filePath)
		if fallbackErr != nil {
			slog.Warn("exiftool fallback failed", "error", fallbackErr)
			return nil, nil
		}
		return data, nil
	}

	if v, err := x.Get(exif.Make); err == nil {
		if s, err := v.StringVal(); err == nil {
			data.CameraMake = &s
		} else {
			s := v.String()
			data.CameraMake = &s
		}
	}
	if v, err := x.Get(exif.Model); err == nil {
		if s, err := v.StringVal(); err == nil {
			data.CameraModel = &s
		} else {
			s := v.String()
			data.CameraModel = &s
		}
	}
	if v, err := x.Get(exif.LensModel); err == nil {
		if s, err := v.StringVal(); err == nil {
			data.LensModel = &s
		} else {
			s := v.String()
			data.LensModel = &s
		}
	}
	if v, err := x.Get(exif.FocalLength); err == nil {
		if n, d, err := v.Rat2(0); err == nil && d != 0 {
			f := float64(n) / float64(d)
			data.FocalLength = &f
		}
	}
	if v, err := x.Get(exif.FNumber); err == nil {
		if n, d, err := v.Rat2(0); err == nil && d != 0 {
			f := float64(n) / float64(d)
			data.Aperture = &f
		}
	}
	if v, err := x.Get(exif.ExposureTime); err == nil {
		if n, d, err := v.Rat2(0); err == nil && d != 0 {
			s := fmt.Sprintf("%d/%d", n, d)
			data.ShutterSpeed = &s
		}
	}
	if v, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if i, err := v.Int(0); err == nil {
			data.ISO = &i
		}
	}
	if v, err := x.Get(exif.DateTimeOriginal); err == nil {
		if s, err := v.StringVal(); err == nil {
			data.DateTaken = &s
		} else {
			s := v.String()
			data.DateTaken = &s
		}
	}
	if lat, lon, err := x.LatLong(); err == nil {
		data.GPSLatitude = &lat
		data.GPSLongitude = &lon
	}
	if v, err := x.Get(exif.Orientation); err == nil {
		if i, err := v.Int(0); err == nil {
			data.Orientation = &i
		}
	}
	if v, err := x.Get(exif.ColorSpace); err == nil {
		if i, err := v.Int(0); err == nil {
			s := "sRGB"
			if i == 0xFFFF {
				s = "Uncalibrated"
			} else if i == 2 {
				s = "Adobe RGB"
			}
			data.ColorSpace = &s
		}
	}
	if v, err := x.Get(exif.Flash); err == nil {
		if i, err := v.Int(0); err == nil {
			data.Flash = &i
		}
	}
	if v, err := x.Get(exif.Software); err == nil {
		if s, err := v.StringVal(); err == nil {
			data.Software = &s
		} else {
			s := v.String()
			data.Software = &s
		}
	}

	data.LensMake = data.CameraMake

	return &data, nil
}

func (s *ExifService) extractViaExifTool(filePath string) (*model.ExifData, error) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		return nil, fmt.Errorf("exiftool not found: %w", err)
	}

	cmd := exec.Command("exiftool", "-json", filePath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("exiftool execution: %w", err)
	}

	var entries []exifToolEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("exiftool json parse: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("exiftool returned no data")
	}

	e := entries[0]
	var data model.ExifData

	if e.Make != "" {
		data.CameraMake = &e.Make
	}
	if e.Model != "" {
		data.CameraModel = &e.Model
	}
	if e.LensModel != "" {
		data.LensModel = &e.LensModel
	}
	if e.LensMake != "" {
		data.LensMake = &e.LensMake
	}
	if e.FocalLength != "" {
		if f, err := parseFocalLength(e.FocalLength); err == nil {
			data.FocalLength = &f
		}
	}
	if e.FNumber > 0 {
		data.Aperture = &e.FNumber
	}
	if e.ExposureTime != "" {
		data.ShutterSpeed = &e.ExposureTime
	}
	if e.ISO > 0 {
		data.ISO = &e.ISO
	}
	if e.DateTimeOriginal != "" {
		data.DateTaken = &e.DateTimeOriginal
	}
	if e.GPSLatitude != 0 || e.GPSLongitude != 0 {
		lat := e.GPSLatitude
		lon := e.GPSLongitude
		data.GPSLatitude = &lat
		data.GPSLongitude = &lon
	}
	if e.GPSAltitude != 0 {
		alt := e.GPSAltitude
		data.GPSAltitude = &alt
	}
	if e.Orientation != 0 {
		ori := e.Orientation
		data.Orientation = &ori
	}
	if e.ColorSpace != "" {
		data.ColorSpace = &e.ColorSpace
	}
	if e.Flash != 0 {
		flash := e.Flash
		data.Flash = &flash
	}
	if e.Software != "" {
		data.Software = &e.Software
	}

	raw, _ := json.Marshal(entries)
	rawStr := string(raw)
	data.RawJSON = &rawStr

	return &data, nil
}

func parseFocalLength(s string) (float64, error) {
	s = trimSuffix(s, " mm")
	s = trimSuffix(s, "mm")
	return strconv.ParseFloat(s, 64)
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}
