package user

import "testing"

func TestValidateUsername(t *testing.T) {
	tests := map[string]bool{
		"alice":                             true,
		"a.b_c-d":                           true,
		"ab":                                false,
		"has space":                         false,
		"emoji🙂name":                        false,
		"alice@x.com":                       false,
		"":                                  false,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": false, // 33 characters
	}
	for username, valid := range tests {
		if err := ValidateUsername(username); (err == nil) != valid {
			t.Errorf("ValidateUsername(%q) error = %v, want valid=%v", username, err, valid)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	tests := map[string]bool{
		"alice@example.com":         true,
		"alice+tag@example.com":     true,
		"not-an-email":              false,
		"Alice <alice@example.com>": false, // display names are not accepted
		"alice@":                    false,
		"":                          false,
	}
	for email, valid := range tests {
		if err := ValidateEmail(email); (err == nil) != valid {
			t.Errorf("ValidateEmail(%q) error = %v, want valid=%v", email, err, valid)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Errorf("NormalizeEmail() = %q", got)
	}
}
