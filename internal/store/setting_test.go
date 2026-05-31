package store

import (
	"testing"
)

func TestSettingStore_SetGet(t *testing.T) {
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSettingStore(db)

	val, err := s.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get missing: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string for missing key, got %q", val)
	}

	if err := s.Set("test_key", "test_value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err = s.Get("test_key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "test_value" {
		t.Errorf("expected test_value, got %q", val)
	}

	if err := s.Set("test_key", "updated_value"); err != nil {
		t.Fatalf("Set update: %v", err)
	}

	val, err = s.Get("test_key")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if val != "updated_value" {
		t.Errorf("expected updated_value, got %q", val)
	}
}

func TestSettingStore_AllowRegistration_default(t *testing.T) {
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSettingStore(db)

	val, err := s.Get("allow_registration")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "false" {
		t.Errorf("expected default allow_registration to be 'false', got %q", val)
	}
}
