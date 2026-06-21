package equipment

import (
	"crypto/rand"
	"math/big"
)

// RollEnhanceSuccess 按目标等级成功率表掷骰，返回 roll(1~100) 与是否成功。
func RollEnhanceSuccess(successRatePct uint32) (rollPct uint32, success bool, err error) {
	if successRatePct > 100 {
		successRatePct = 100
	}
	draw, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return 0, false, err
	}
	rollPct = uint32(draw.Int64()) + 1
	return rollPct, rollPct <= successRatePct, nil
}
