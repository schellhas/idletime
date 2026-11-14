package password

import (
	"errors"
	"strings"
	"testing"

	"github.com/schellhas/idletime/backend/internal/validation"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "too short", password: "short", wantErr: true},
		{name: "minimum length", password: strings.Repeat("a", MinLength)},
		{name: "maximum length", password: strings.Repeat("a", maxLength)},
		{name: "too long", password: strings.Repeat("a", maxLength+1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, validation.ErrInvalid) {
				t.Errorf("error %v does not wrap validation.ErrInvalid", err)
			}
		})
	}
}

func TestHashAndMatches(t *testing.T) {
	hash, err := Hash("correct horse battery")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "correct horse battery" {
		t.Fatal("hash must not equal the plain password")
	}
	if !Matches(hash, "correct horse battery") {
		t.Error("expected the correct password to match")
	}
	if Matches(hash, "wrong password") {
		t.Error("expected a wrong password not to match")
	}
}
