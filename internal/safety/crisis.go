package safety

import "strings"

// Phrases chosen for high precision over recall — the model's own safety
// instructions cover subtler cases; this is the hard tripwire.
var crisisPhrases = []string{
	"kill myself",
	"killing myself",
	"end my life",
	"ending my life",
	"end it all",
	"take my own life",
	"suicide",
	"suicidal",
	"don't want to be alive",
	"dont want to be alive",
	"don't want to live",
	"dont want to live",
	"better off dead",
	"better off without me",
	"no reason to live",
	"want to die",
	"wanna die",
	"wish i was dead",
	"wish i were dead",
	"hurt myself",
	"hurting myself",
	"harm myself",
	"self-harm",
	"self harm",
	"cutting myself",
	"overdose on",
	"plan to od",
}

// DetectCrisis reports whether the text contains high-risk language.
func DetectCrisis(text string) bool {
	lower := strings.ToLower(text)
	for _, phrase := range crisisPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}
