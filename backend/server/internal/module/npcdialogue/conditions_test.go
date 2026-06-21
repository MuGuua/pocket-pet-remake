package npcdialogue

import (
	"context"
	"encoding/json"
	"testing"

	"pocket-pet-remake/server/internal/module/quest"
)

type stubQuestReader struct {
	summaries []quest.Summary
}

func (s stubQuestReader) ListSummaries(_ context.Context, _ uint64) ([]quest.Summary, error) {
	return s.summaries, nil
}

func TestMatchNodeConditionsQuestState(t *testing.T) {
	reader := stubQuestReader{summaries: []quest.Summary{{QuestID: 1001, State: quest.StateAccepted}}}
	ok, err := MatchNodeConditions(context.Background(), reader, 1, json.RawMessage(`{"quest_id":1001,"quest_state":"accepted"}`))
	if err != nil {
		t.Fatalf("MatchNodeConditions() error = %v", err)
	}
	if !ok {
		t.Fatal("MatchNodeConditions() = false, want true")
	}
}

func TestMatchNodeConditionsQuestStateMismatch(t *testing.T) {
	reader := stubQuestReader{summaries: []quest.Summary{{QuestID: 1001, State: quest.StateAvailable}}}
	ok, err := MatchNodeConditions(context.Background(), reader, 1, json.RawMessage(`{"quest_id":1001,"quest_state":"completed"}`))
	if err != nil {
		t.Fatalf("MatchNodeConditions() error = %v", err)
	}
	if ok {
		t.Fatal("MatchNodeConditions() = true, want false")
	}
}
