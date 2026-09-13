package credentials

import (
	"errors"
	"strings"
	"testing"
)

func TestHashVerify(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("hash is not a PHC argon2id string: %q", hash)
	}
	if err := VerifyPassword("correct-horse", hash); err != nil {
		t.Errorf("VerifyPassword with correct password: %v", err)
	}
	if err := VerifyPassword("wrong", hash); !errors.Is(err, ErrPasswordMismatch) {
		t.Errorf("VerifyPassword with wrong password = %v, want ErrPasswordMismatch", err)
	}
}

func TestHashUsesFreshSalt(t *testing.T) {
	a, err := HashPassword("same")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	b, err := HashPassword("same")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if a == b {
		t.Error("identical passwords produced identical hashes; salt is not random")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		encoded string
	}{
		{"empty", ""},
		{"not phc", "plaintext"},
		{"bcrypt", "$2a$10$abcdefghijklmnopqrstuv"},
		{"truncated", "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyPassword("x", tc.encoded)
			if err == nil {
				t.Fatal("expected an error")
			}
			if errors.Is(err, ErrPasswordMismatch) {
				t.Error("malformed hash reported as ErrPasswordMismatch; should be a parse error")
			}
		})
	}
}
