package model

type ExifData struct {
	FileID       string   `json:"fileId" db:"file_id"`
	CameraMake   *string  `json:"cameraMake,omitempty" db:"camera_make"`
	CameraModel  *string  `json:"cameraModel,omitempty" db:"camera_model"`
	LensMake     *string  `json:"lensMake,omitempty" db:"lens_make"`
	LensModel    *string  `json:"lensModel,omitempty" db:"lens_model"`
	FocalLength  *float64 `json:"focalLength,omitempty" db:"focal_length"`
	Aperture     *float64 `json:"aperture,omitempty" db:"aperture"`
	ShutterSpeed *string  `json:"shutterSpeed,omitempty" db:"shutter_speed"`
	ISO          *int     `json:"iso,omitempty" db:"iso"`
	DateTaken    *string  `json:"dateTaken,omitempty" db:"date_taken"`
	GPSLatitude  *float64 `json:"gpsLatitude,omitempty" db:"gps_latitude"`
	GPSLongitude *float64 `json:"gpsLongitude,omitempty" db:"gps_longitude"`
	GPSAltitude  *float64 `json:"gpsAltitude,omitempty" db:"gps_altitude"`
	Orientation  *int     `json:"orientation,omitempty" db:"orientation"`
	ColorSpace   *string  `json:"colorSpace,omitempty" db:"color_space"`
	Flash        *int     `json:"flash,omitempty" db:"flash"`
	Software     *string  `json:"software,omitempty" db:"software"`
	RawJSON      *string  `json:"rawJson,omitempty" db:"raw_json"`
}
