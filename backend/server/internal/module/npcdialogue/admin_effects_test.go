package npcdialogue

import (
	"encoding/json"
	"testing"
)

func TestEncodeDecodeAdminEffectsJSON(t *testing.T) {
	raw := EncodeAdminEffectsJSON(AdminDialogueEffects{Notice: "你好", QuestEvent: "TALK_TO_NPC"})
	decoded := DecodeAdminEffectsJSON(raw)
	if decoded.Notice != "你好" || decoded.QuestEvent != "TALK_TO_NPC" {
		t.Fatalf("DecodeAdminEffectsJSON() = %#v, want notice=你好 quest_event=TALK_TO_NPC", decoded)
	}
}

func TestEncodeAdminEffectsJSONPartialAndEmpty(t *testing.T) {
	noticeOnly := EncodeAdminEffectsJSON(AdminDialogueEffects{Notice: "提示文案"})
	if string(noticeOnly) != `{"notice":"提示文案"}` {
		t.Fatalf("EncodeAdminEffectsJSON(notice only) = %s", string(noticeOnly))
	}
	if string(EncodeAdminEffectsJSON(AdminDialogueEffects{})) != "{}" {
		t.Fatalf("EncodeAdminEffectsJSON(empty) should be {}")
	}
	decoded := DecodeAdminEffectsJSON(json.RawMessage(`{"quest_event":"TALK_TO_NPC"}`))
	if decoded.QuestEvent != "TALK_TO_NPC" || decoded.Notice != "" {
		t.Fatalf("DecodeAdminEffectsJSON() = %#v", decoded)
	}
}
