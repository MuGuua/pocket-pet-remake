package wstransport

import "strings"

// resolveDialogueSpeaker 把后台/数据库中的说话人占位符解析成客户端可直接展示的值。
// 返回值中的 isPlayerSpeaker 供客户端决定角标显示在右上还是左上，避免仅靠名字猜测。
func resolveDialogueSpeaker(rawSpeaker string, rawPortraitKey string, playerName string) (speaker string, portraitKey string, isPlayerSpeaker bool) {
	normalizedSpeaker := strings.TrimSpace(rawSpeaker)
	normalizedPortrait := strings.TrimSpace(rawPortraitKey)
	resolvedPlayerName := strings.TrimSpace(playerName)
	if resolvedPlayerName == "" {
		resolvedPlayerName = "训练家"
	}

	if isPlayerSpeakerToken(normalizedSpeaker) {
		portrait := normalizedPortrait
		if portrait == "" || portrait == "default" {
			portrait = "player_default"
		}
		return resolvedPlayerName, portrait, true
	}

	if normalizedPortrait == "player_default" {
		if normalizedSpeaker == "" {
			return resolvedPlayerName, "player_default", true
		}
		return normalizedSpeaker, "player_default", true
	}

	return normalizedSpeaker, normalizedPortrait, false
}

// isPlayerSpeakerToken 判断配置层是否把当前台词标记为玩家说话。
func isPlayerSpeakerToken(speaker string) bool {
	switch strings.TrimSpace(speaker) {
	case "@player", "$player", "玩家", "{player_name}":
		return true
	default:
		return false
	}
}
