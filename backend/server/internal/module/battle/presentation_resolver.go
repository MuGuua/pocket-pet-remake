package battle

import (
	"strings"

	"pocket-pet-remake/server/internal/module/player"
)

// DefaultCharacterSkinID 是人物单位在尚未配置形象时的全局默认战斗皮肤。
const DefaultCharacterSkinID = player.DefaultPlayerSkinID

// petSkinResolver 由 bootstrap 注入，按 pet_id 查询 pet_definition.skin_id。
var petSkinResolver func(uint32) string

// SetPetSkinResolver 注册宠物模板外观解析器；未注入时宠物 skin_id 为空字符串。
func SetPetSkinResolver(resolver func(uint32) string) {
	petSkinResolver = resolver
}

func resolvePetSkinID(petID uint32) string {
	if petID == 0 || petSkinResolver == nil {
		return ""
	}
	return strings.TrimSpace(petSkinResolver(petID))
}

func resolveActorSkinID(actor *actorRuntime) string {
	if actor == nil {
		return ""
	}
	switch actor.unitClass {
	case ActorUnitClassCharacter:
		skinID := strings.TrimSpace(actor.skinID)
		if skinID != "" {
			return skinID
		}
		return DefaultCharacterSkinID
	case ActorUnitClassPet:
		return resolvePetSkinID(actor.petID)
	case ActorUnitClassMonster:
		return strings.TrimSpace(actor.skinID)
	default:
		return ""
	}
}
