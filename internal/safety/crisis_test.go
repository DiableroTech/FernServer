package safety

import "testing"

func TestDetectCrisis(t *testing.T) {
	positive := []string{
		"I just want to end my life",
		"sometimes i think about KILLING MYSELF",
		"I've been having suicidal thoughts again",
		"honestly everyone would be better off without me",
		"i keep cutting myself when it gets bad",
	}
	for _, s := range positive {
		if !DetectCrisis(s) {
			t.Errorf("expected crisis detection for %q", s)
		}
	}

	negative := []string{
		"this deadline is killing me lol",
		"I had a rough day at work",
		"my plants are dying and it makes me sad",
		"I'm exhausted and want this week to end",
	}
	for _, s := range negative {
		if DetectCrisis(s) {
			t.Errorf("false positive for %q", s)
		}
	}
}
