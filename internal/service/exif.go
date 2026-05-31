package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os/exec"
	"strconv"

	jpegexif "github.com/dsoprea/go-exif/v3"
	"github.com/dsoprea/go-exif/v3/common"
	"github.com/dsoprea/go-jpeg-image-structure/v2"

	"github.com/drive/drive/internal/model"
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
	data, err := s.extractViaDsoprea(filePath)
	if err == nil && data != nil {
		return data, nil
	}

	data, fallbackErr := s.extractViaExifTool(filePath)
	if fallbackErr != nil {
		slog.Warn("exiftool fallback failed", "error", fallbackErr)
		return nil, nil
	}
	return data, nil
}

func (s *ExifService) extractViaDsoprea(filePath string) (*model.ExifData, error) {
	jmp := jpegstructure.NewJpegMediaParser()
	mc, err := jmp.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("jpeg parse: %w", err)
	}

	rootIfd, _, err := mc.Exif()
	if err != nil {
		return nil, fmt.Errorf("exif extract: %w", err)
	}

	var data model.ExifData

	if results, err := rootIfd.FindTagWithName("Make"); err == nil && len(results) > 0 {
		s := valueString(results[0])
		data.CameraMake = &s
	}
	if results, err := rootIfd.FindTagWithName("Model"); err == nil && len(results) > 0 {
		s := valueString(results[0])
		data.CameraModel = &s
	}
	if results, err := rootIfd.FindTagWithName("Orientation"); err == nil && len(results) > 0 {
		if i, err := results[0].Value(); err == nil {
			if arr, ok := i.([]uint16); ok && len(arr) > 0 {
				v := int(arr[0])
				data.Orientation = &v
			}
		}
	}
	if results, err := rootIfd.FindTagWithName("Software"); err == nil && len(results) > 0 {
		s := valueString(results[0])
		data.Software = &s
	}

	exifIfd, err := jpegexif.FindIfdFromRootIfd(rootIfd, "IFD/Exif")
	if err == nil && exifIfd != nil {
		if results, err := exifIfd.FindTagWithName("LensModel"); err == nil && len(results) > 0 {
			s := valueString(results[0])
			data.LensModel = &s
		}
		if results, err := exifIfd.FindTagWithName("FocalLength"); err == nil && len(results) > 0 {
			if f := rationalFloat(results[0]); f != nil {
				data.FocalLength = f
			}
		}
		if results, err := exifIfd.FindTagWithName("FNumber"); err == nil && len(results) > 0 {
			if f := rationalFloat(results[0]); f != nil {
				data.Aperture = f
			}
		}
		if results, err := exifIfd.FindTagWithName("ExposureTime"); err == nil && len(results) > 0 {
			if s := rationalString(results[0]); s != "" {
				data.ShutterSpeed = &s
			}
		}
		if results, err := exifIfd.FindTagWithName("ISOSpeedRatings"); err == nil && len(results) > 0 {
			if i, err := results[0].Value(); err == nil {
				if arr, ok := i.([]uint16); ok && len(arr) > 0 {
					v := int(arr[0])
					data.ISO = &v
				}
			}
		}
		if results, err := exifIfd.FindTagWithName("DateTimeOriginal"); err == nil && len(results) > 0 {
			s := valueString(results[0])
			data.DateTaken = &s
		}
		if results, err := exifIfd.FindTagWithName("ColorSpace"); err == nil && len(results) > 0 {
			if i, err := results[0].Value(); err == nil {
				if arr, ok := i.([]uint16); ok && len(arr) > 0 {
					s := "sRGB"
					if arr[0] == 0xFFFF {
						s = "Uncalibrated"
					} else if arr[0] == 2 {
						s = "Adobe RGB"
					}
					data.ColorSpace = &s
				}
			}
		}
		if results, err := exifIfd.FindTagWithName("Flash"); err == nil && len(results) > 0 {
			if i, err := results[0].Value(); err == nil {
				if arr, ok := i.([]uint16); ok && len(arr) > 0 {
					v := int(arr[0])
					data.Flash = &v
				}
			}
		}
	}

	if gi, err := rootIfd.GpsInfo(); err == nil && gi != nil {
		lat := math.Round(gi.Latitude.Decimal()*1e7) / 1e7
		lon := math.Round(gi.Longitude.Decimal()*1e7) / 1e7
		data.GPSLatitude = &lat
		data.GPSLongitude = &lon
	}

	data.LensMake = data.CameraMake

	return &data, nil
}

func valueString(ite *jpegexif.IfdTagEntry) string {
	v, err := ite.Value()
	if err != nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	s, err := ite.FormatFirst()
	if err != nil {
		return ""
	}
	return s
}

func rationalFloat(ite *jpegexif.IfdTagEntry) *float64 {
	v, err := ite.Value()
	if err != nil {
		return nil
	}
	rat, ok := v.([]exifcommon.Rational)
	if !ok || len(rat) == 0 {
		return nil
	}
	if rat[0].Denominator == 0 {
		return nil
	}
	f := float64(rat[0].Numerator) / float64(rat[0].Denominator)
	return &f
}

func rationalString(ite *jpegexif.IfdTagEntry) string {
	v, err := ite.Value()
	if err != nil {
		return ""
	}
	rat, ok := v.([]exifcommon.Rational)
	if !ok || len(rat) == 0 {
		return ""
	}
	if rat[0].Denominator == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", rat[0].Numerator, rat[0].Denominator)
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
