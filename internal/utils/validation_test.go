package utils

import (
	"errors"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "trims spaces", input: "  alice  ", want: "alice"},
		{name: "keeps inner spaces", input: "alice smith", want: "alice smith"},
		{name: "empty", input: "", wantErr: ErrEmptyName},
		{name: "only spaces", input: "   ", wantErr: ErrEmptyName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeName(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NormalizeName(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("NormalizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	t.Run("nil becomes empty slice", func(t *testing.T) {
		got, err := NormalizeTags(nil)
		if err != nil {
			t.Fatalf("NormalizeTags(nil) error = %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("NormalizeTags(nil) = %#v, want empty non-nil slice", got)
		}
	})

	t.Run("copies input", func(t *testing.T) {
		input := []string{"vip"}
		got, err := NormalizeTags(input)
		if err != nil {
			t.Fatalf("NormalizeTags() error = %v", err)
		}
		got[0] = "changed"
		if input[0] != "vip" {
			t.Fatalf("NormalizeTags() modified input: %#v", input)
		}
	})

	t.Run("valid tags", func(t *testing.T) {
		got, err := NormalizeTags([]string{"vip", "new"})
		if err != nil {
			t.Fatalf("NormalizeTags() error = %v", err)
		}
		if len(got) != 2 || got[0] != "vip" || got[1] != "new" {
			t.Fatalf("NormalizeTags() = %#v", got)
		}
	})

	t.Run("empty tag", func(t *testing.T) {
		_, err := NormalizeTags([]string{"vip", ""})
		if !errors.Is(err, ErrEmptyTag) {
			t.Fatalf("NormalizeTags() error = %v, want %v", err, ErrEmptyTag)
		}
	})
}
