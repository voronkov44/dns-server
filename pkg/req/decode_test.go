package req

import (
	"errors"
	"strings"
	"testing"
)

type decodeTestPayload struct {
	Server string `json:"server"`
}

func TestDecodeReturnsErrEmptyBody(t *testing.T) {
	_, err := Decode[decodeTestPayload](strings.NewReader(""))
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("Decode() error = %v, want %v", err, ErrEmptyBody)
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	_, err := Decode[decodeTestPayload](strings.NewReader(`{"server":"8.8.8.8","extra":true}`))
	if err == nil {
		t.Fatal("Decode() error = nil, want unknown field error")
	}
}

func TestDecodeRejectsMultipleJSONObjects(t *testing.T) {
	_, err := Decode[decodeTestPayload](strings.NewReader(`{"server":"8.8.8.8"}{"server":"1.1.1.1"}`))
	if err == nil {
		t.Fatal("Decode() error = nil, want multiple JSON values error")
	}
}
