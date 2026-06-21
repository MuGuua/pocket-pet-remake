package npcdialogue

import (
	"encoding/json"
	"testing"
)

func TestEncodeDecodeAdminConditionsJSON(t *testing.T) {
	raw := EncodeAdminConditionsJSON(AdminDialogueConditions{QuestID: 1001, QuestState: "ACCEPTED"})
	decoded := DecodeAdminConditionsJSON(raw)
	if decoded.QuestID != 1001 || decoded.QuestState != "ACCEPTED" {
		t.Fatalf("DecodeAdminConditionsJSON() = %#v, want quest 1001 ACCEPTED", decoded)
	}
}

func TestEncodeAdminConditionsJSONEmpty(t *testing.T) {
	raw := EncodeAdminConditionsJSON(AdminDialogueConditions{})
	if string(raw) != "{}" {
		t.Fatalf("EncodeAdminConditionsJSON(empty) = %s, want {}", string(raw))
	}
	decoded := DecodeAdminConditionsJSON(json.RawMessage(`{"quest_id":1002,"quest_state":"AVAILABLE"}`))
	if decoded.QuestID != 1002 || decoded.QuestState != "AVAILABLE" {
		t.Fatalf("DecodeAdminConditionsJSON() = %#v", decoded)
	}
}
