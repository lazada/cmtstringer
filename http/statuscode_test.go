package http

import (
	"fmt"
	"testing"
)

func TestStatusCodeMessage(t *testing.T) {
	data := map[StatusCode]string{
		StatusBadRequest: "Bad Request",
		StatusNotFound:   "Not Found",
	}

	for code, msg := range data {
		t.Run(msg, func(t *testing.T) {
			assertEqual(t, fmt.Sprintf("%v", code), msg)
		})
	}
}

func assertEqual(t *testing.T, actual, expected string) {
	if actual != expected {
		t.Fatalf("HTTP StatusCode message is incorrect\nExpected: %s\nObtained: %s", expected, actual)
	}
}
