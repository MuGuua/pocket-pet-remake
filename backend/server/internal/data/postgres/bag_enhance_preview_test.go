package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"pocket-pet-remake/server/internal/module/bag"
)

// 验证同一份材料持有量用于多个装备时，装备专属预览互不覆盖，也不污染共享输入。
func TestEnhancePreviewOwnsMaterialOptions(t *testing.T) {
	materials := []bag.RuntimeEnhanceMaterialOption{{ItemID: 1, OwnedQuantity: 3}}
	first, err := buildRuntimeEnhancePreview(context.Background(), nil, 1, materials, 1, false, 0, 0, "", "", false, 0, 0, 0, 0, 0, nil, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildRuntimeEnhancePreview(context.Background(), nil, 1, materials, 2, false, 0, 0, "", "", false, 0, 0, 0, 0, 0, nil, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	first.Materials[0].EffectiveSuccessRatePct = 70
	first.Materials[0].FailurePenalty = "downgrade"
	if second.Materials[0].EffectiveSuccessRatePct != 0 || second.Materials[0].FailurePenalty != "" {
		t.Fatalf("equipment previews share mutable options: %+v", second.Materials)
	}
	if materials[0].EffectiveSuccessRatePct != 0 || materials[0].FailurePenalty != "" {
		t.Fatalf("shared material quantities were changed: %+v", materials)
	}
	if first.Materials[0].OwnedQuantity != 3 || second.Materials[0].OwnedQuantity != 3 {
		t.Fatal("material quantities must be preserved")
	}
}

// 没有强化材料时保留空数组协议，避免客户端把 null 误当成缺失的材料快照。
func TestEnhancePreviewEmptyMaterialsRemainArray(t *testing.T) {
	preview, err := buildRuntimeEnhancePreview(context.Background(), nil, 1, nil, 1, false, 0, 0, "", "", false, 0, 0, 0, 0, 0, nil, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(preview)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"materials":[]`) {
		t.Fatalf("empty material array changed shape: %s", encoded)
	}
}
