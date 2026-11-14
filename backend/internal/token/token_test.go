package token

import "testing"

func TestNew(t *testing.T) {
	rawA, hashA, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	rawB, _, _ := New()

	if rawA == rawB {
		t.Error("tokens must be unique")
	}
	if rawA == hashA {
		t.Error("hash must differ from the raw token")
	}
	if Hash(rawA) != hashA {
		t.Error("Hash(raw) must equal the hash returned by New")
	}
}
