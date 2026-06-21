package wstransport

import "testing"

func TestResolveDialogueSpeakerPlayerToken(t *testing.T) {
	speaker, portraitKey, isPlayerSpeaker := resolveDialogueSpeaker("玩家", "", "李白100")
	if !isPlayerSpeaker {
		t.Fatalf("expected player speaker")
	}
	if speaker != "李白100" {
		t.Fatalf("speaker = %q, want 李白100", speaker)
	}
	if portraitKey != "player_default" {
		t.Fatalf("portraitKey = %q, want player_default", portraitKey)
	}
}

func TestResolveDialogueSpeakerAtPlayerToken(t *testing.T) {
	speaker, portraitKey, isPlayerSpeaker := resolveDialogueSpeaker("@player", "player_default", "Demo")
	if !isPlayerSpeaker {
		t.Fatalf("expected player speaker")
	}
	if speaker != "Demo" {
		t.Fatalf("speaker = %q, want Demo", speaker)
	}
	if portraitKey != "player_default" {
		t.Fatalf("portraitKey = %q, want player_default", portraitKey)
	}
}

func TestResolveDialogueSpeakerNPC(t *testing.T) {
	speaker, portraitKey, isPlayerSpeaker := resolveDialogueSpeaker("市场理萌", "npc_limeng_normal", "李白100")
	if isPlayerSpeaker {
		t.Fatalf("did not expect player speaker")
	}
	if speaker != "市场理萌" {
		t.Fatalf("speaker = %q, want 市场理萌", speaker)
	}
	if portraitKey != "npc_limeng_normal" {
		t.Fatalf("portraitKey = %q, want npc_limeng_normal", portraitKey)
	}
}

func TestResolveDialogueSpeakerPortraitOnly(t *testing.T) {
	speaker, portraitKey, isPlayerSpeaker := resolveDialogueSpeaker("", "player_default", "李白100")
	if !isPlayerSpeaker {
		t.Fatalf("expected player speaker from portrait key")
	}
	if speaker != "李白100" {
		t.Fatalf("speaker = %q, want 李白100", speaker)
	}
	if portraitKey != "player_default" {
		t.Fatalf("portraitKey = %q, want player_default", portraitKey)
	}
}
