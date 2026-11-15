package password

import "testing"

func TestHashRejectsEmptyPassword(t *testing.T) {
	svc := New()
	if _, err := svc.Hash(""); err == nil {
		t.Fatal("Hash(\"\") returned nil error")
	}
}

func TestHashAndVerify(t *testing.T) {
	svc := New()

	hash, err := svc.Hash("s3cret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "" {
		t.Fatal("Hash() returned empty string")
	}

	valid, err := svc.Verify(hash, "s3cret")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !valid {
		t.Fatal("Verify() = false for correct password")
	}

	valid, err = svc.Verify(hash, "wrong")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if valid {
		t.Fatal("Verify() = true for wrong password")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	svc := New()

	tests := []string{"", "not-base64", "c2hvcnQ"}
	for _, encoded := range tests {
		valid, err := svc.Verify(encoded, "s3cret")
		if err != nil {
			t.Fatalf("Verify(%q) error = %v", encoded, err)
		}
		if valid {
			t.Fatalf("Verify(%q) = true, want false", encoded)
		}
	}
}

func TestHashUsesRandomSalt(t *testing.T) {
	svc := New()

	first, err := svc.Hash("same-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	second, err := svc.Hash("same-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if first == second {
		t.Fatal("Hash() produced identical hashes for the same password")
	}
}
