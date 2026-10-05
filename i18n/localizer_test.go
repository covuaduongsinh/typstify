package i18n_test

import (
	"testing"

	"looz.ws/typstify/i18n"
)

func TestVietnameseLocale(t *testing.T) {
	err := i18n.SetLocale("vi-vn")
	if err != nil {
		t.Fatalf("SetLocale vi-vn failed: %v", err)
	}

	cases := []struct {
		in   string
		want string
	}{
		{"Settings", "Cài đặt"},
		{"Sync", "Đồng bộ"},
		{"AI Assistant", "Trợ lý AI"},
		{"Save", "Lưu"},
		{"Export", "Xuất bản"},
	}

	for _, c := range cases {
		got := i18n.Translate(c.in)
		if got != c.want {
			t.Errorf("Translate(%q) in vi-VN = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEnglishLocale(t *testing.T) {
	err := i18n.SetLocale("en-US")
	if err != nil {
		t.Fatalf("SetLocale en-US failed: %v", err)
	}

	cases := []struct {
		in   string
		want string
	}{
		{"Settings", "Settings"},
		{"Sync", "Sync"},
		{"AI Assistant", "AI Assistant"},
		{"Save", "Save"},
	}

	for _, c := range cases {
		got := i18n.Translate(c.in)
		if got != c.want {
			t.Errorf("Translate(%q) in en-US = %q, want %q", c.in, got, c.want)
		}
	}
}
