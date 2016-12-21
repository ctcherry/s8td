package main

import (
	"testing"
	"time"
)

func TestValidateTimestamp(t *testing.T) {
	goodTs := time.Now().Unix() - 10
	badPastTs := time.Now().Unix() - 100
	badFutureTs := time.Now().Unix() + 100

	if !validateTimestamp(goodTs) {
		t.Error("expected goodTs to pass validateTimestamp")
	}

	if validateTimestamp(badPastTs) {
		t.Error("expected badPastTs to fail validateTimestamp")
	}

	if validateTimestamp(badFutureTs) {
		t.Error("expected badFutureTs to fail validateTimestamp")
	}
}
