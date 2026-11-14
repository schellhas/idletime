package optional

import (
	"encoding/json"
	"testing"
)

func TestNullable(t *testing.T) {
	type patch struct {
		Parent Nullable[int64] `json:"parent"`
	}

	tests := []struct {
		name      string
		json      string
		wantSet   bool
		wantValue *int64
	}{
		{name: "absent", json: `{}`},
		{name: "null", json: `{"parent":null}`, wantSet: true},
		{name: "value", json: `{"parent":7}`, wantSet: true, wantValue: ptr(7)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p patch
			if err := json.Unmarshal([]byte(tt.json), &p); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if p.Parent.Set != tt.wantSet {
				t.Errorf("Set = %v, want %v", p.Parent.Set, tt.wantSet)
			}
			if (p.Parent.Value == nil) != (tt.wantValue == nil) || (tt.wantValue != nil && *p.Parent.Value != *tt.wantValue) {
				t.Errorf("Value = %v, want %v", p.Parent.Value, tt.wantValue)
			}
		})
	}

	var p patch
	if err := json.Unmarshal([]byte(`{"parent":"x"}`), &p); err == nil {
		t.Error("expected an error for a value of the wrong type")
	}
}

func ptr(v int64) *int64 { return &v }
