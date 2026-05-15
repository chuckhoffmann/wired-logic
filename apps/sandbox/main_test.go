package main

import (
	"testing"
)

func TestKeyRepeatTriggered(t *testing.T) {
	tests := []struct {
		name       string
		pressCount int
		want       bool
	}{
		{name: "negative count", pressCount: -1, want: false},
		{name: "initial press triggers", pressCount: 0, want: true},
		{name: "before initial delay does not trigger", pressCount: 1, want: false},
		{name: "just before delay end does not trigger", pressCount: cursorInitialDelayTicks - 1, want: false},
		{name: "at delay boundary triggers", pressCount: cursorInitialDelayTicks, want: true},
		{name: "between repeat ticks does not trigger", pressCount: cursorInitialDelayTicks + 1, want: false},
		{name: "next repeat tick triggers", pressCount: cursorInitialDelayTicks + cursorRepeatTicks, want: true},
		{name: "later repeat tick triggers", pressCount: cursorInitialDelayTicks + 2*cursorRepeatTicks, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := keyRepeatTriggered(tc.pressCount)
			if got != tc.want {
				t.Fatalf("keyRepeatTriggered(%d) = %v, want %v", tc.pressCount, got, tc.want)
			}
		})
	}
}
