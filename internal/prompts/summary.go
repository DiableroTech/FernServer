package prompts

import (
	"fmt"
	"strings"
)

const SummarySystem = `You are Fern, an AI journaling companion. You will be given the transcript of a journaling session you just had with a person. Produce a session summary for them.

Respond with ONLY a JSON object, no markdown fences, in exactly this shape:
{
  "summary": "2-4 sentences capturing what the session was about, written warmly and in second person ('You explored...'). Specific, not generic.",
  "insights": ["2-4 short insight strings — patterns noticed, reframes found, or things worth sitting with. Each one specific to this session."],
  "moodLabel": "one or two words for the dominant emotional tone, e.g. 'anxious but hopeful'",
  "moodScore": 6
}

moodScore is an integer 1-10 rating the person's overall emotional state during the session: 1-3 = struggling/distressed, 4-5 = low or flat, 6-7 = steady/okay, 8-10 = genuinely good. Judge from their words, not from politeness. If the session ended in a better place than it started, lean toward where they ended.`

// MemoryContext formats prior session summaries for injection into the
// system prompt.
func MemoryContext(entries []MemoryEntry) string {
	if len(entries) == 0 {
		return ""
	}
	out := "\n\nWhat you remember about this person from recent journaling sessions (most recent first). Draw on this naturally when relevant — reference past themes the way a good therapist would, without reciting it back mechanically:\n"
	for _, e := range entries {
		out += fmt.Sprintf("- [%s, %s session, mood: %s] %s\n", e.Date, e.Modality, e.MoodLabel, e.Summary)
	}
	return out
}

type MemoryEntry struct {
	Date      string
	Modality  string
	MoodLabel string
	Summary   string
}

// RecoveryContext formats the person's recovery state for the system prompt.
func RecoveryContext(focus string, streakDays, cravingsThisWeek int, recentTriggers []string) string {
	out := fmt.Sprintf("\n\nThis person is in recovery from %s. Current streak: %d day(s). Cravings logged this week: %d.", focus, streakDays, cravingsThisWeek)
	if len(recentTriggers) > 0 {
		out += " Known triggers: " + strings.Join(recentTriggers, ", ") + "."
	}
	out += " Hold this with care: acknowledge milestones specifically when relevant, never moralize about lapses, and treat cravings as information rather than failure. Don't bring recovery up unprompted every turn — but be aware of it."
	return out
}
