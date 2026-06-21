package quest

import "testing"

// 验证多阶段任务只会推进当前最早未完成阶段，避免跳阶段完成后续目标。
func TestApplyEventToObjectivesSequentialStages(t *testing.T) {
	template := Template{
		QuestID: 9001,
		Objectives: []ObjectiveTemplate{
			{ObjectiveID: 1, EventType: "TALK_TO_NPC", Description: "阶段1", TargetValue: 1, TargetSelector: map[string]any{"npc_id": uint64(93001)}},
			{ObjectiveID: 2, EventType: "TALK_TO_NPC", Description: "阶段2", TargetValue: 1, TargetSelector: map[string]any{"npc_id": uint64(93002)}},
		},
	}
	objectives := []PlayerObjective{
		{ObjectiveID: 1, Description: "阶段1", CurrentValue: 0, TargetValue: 1, Completed: false},
		{ObjectiveID: 2, Description: "阶段2", CurrentValue: 0, TargetValue: 1, Completed: false},
	}

	updated, changed := applyEventToObjectives(template, objectives, Event{
		EventType: "TALK_TO_NPC",
		NPCID:     93002,
		Count:     1,
	})
	if changed {
		t.Fatal("stage 2 event should be ignored before stage 1 completes")
	}
	if updated[0].Completed || updated[0].CurrentValue != 0 {
		t.Fatalf("stage 1 should stay incomplete, got current=%d completed=%v", updated[0].CurrentValue, updated[0].Completed)
	}
	if updated[1].Completed || updated[1].CurrentValue != 0 {
		t.Fatalf("stage 2 should not progress yet, got current=%d completed=%v", updated[1].CurrentValue, updated[1].Completed)
	}

	updated, changed = applyEventToObjectives(template, objectives, Event{
		EventType: "TALK_TO_NPC",
		NPCID:     93001,
		Count:     1,
	})
	if !changed {
		t.Fatal("expected stage 1 to progress")
	}
	if !updated[0].Completed {
		t.Fatal("stage 1 should be completed")
	}

	updated, changed = applyEventToObjectives(template, updated, Event{
		EventType: "TALK_TO_NPC",
		NPCID:     93002,
		Count:     1,
	})
	if !changed {
		t.Fatal("expected stage 2 to progress after stage 1 completes")
	}
	if !updated[1].Completed {
		t.Fatal("stage 2 should be completed")
	}
}
