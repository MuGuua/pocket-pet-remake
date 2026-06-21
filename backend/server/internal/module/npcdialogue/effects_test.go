package npcdialogue

import (
	"encoding/json"
	"testing"
)

func TestParseNodeEffects(t *testing.T) {
	effects := ParseNodeEffects(json.RawMessage(`{"notice":"你好","quest_event":"TALK_TO_NPC"}`))
	if effects.Notice != "你好" {
		t.Fatalf("Notice = %q, want 你好", effects.Notice)
	}
	if effects.QuestEvent != "TALK_TO_NPC" {
		t.Fatalf("QuestEvent = %q, want TALK_TO_NPC", effects.QuestEvent)
	}
}

func TestParseNodeEffectsEmpty(t *testing.T) {
	effects := ParseNodeEffects(json.RawMessage(`{}`))
	if effects.Notice != "" || effects.QuestEvent != "" {
		t.Fatalf("ParseNodeEffects({}) = %#v, want empty", effects)
	}
}
