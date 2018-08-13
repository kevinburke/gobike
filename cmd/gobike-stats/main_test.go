package main

import (
	"testing"
	"time"
)

func TestBefore(t *testing.T) {
	d := Day{Day: 30, Month: time.February, Year: 2005}
	d2 := Day{Day: 5, Month: time.January, Year: 2006}
	if !d.Before(d2) {
		t.Errorf("d should be before d2")
	}
	if d2.Before(d) {
		t.Errorf("d2 should not be before d")
	}
}
