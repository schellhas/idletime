// Package optional distinguishes "field absent" from "field set to null" in JSON patches.
package optional

import "encoding/json"

// Nullable is a JSON value that records whether it was present at all.
//
//	absent:  Set == false
//	null:    Set == true, Value == nil
//	value:   Set == true, Value != nil
type Nullable[T any] struct {
	Set   bool
	Value *T
}

// UnmarshalJSON implements json.Unmarshaler. It is only called when the key is present.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.Value = &value
	return nil
}
