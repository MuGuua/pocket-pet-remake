package equipment

import "fmt"

const (
	// EnhanceRequiredLevelBandSize 穿戴等级段宽度（每10级一段）。
	EnhanceRequiredLevelBandSize uint32 = 10
	// MaxEnhanceRequiredLevelBandMin 后台可配置的最大穿戴等级段起点（101=101~110级）。
	MaxEnhanceRequiredLevelBandMin uint32 = 101
)

// ResolveRequiredLevelBandMin 根据装备穿戴等级解析所属等级段起点（1/11/21…）。
func ResolveRequiredLevelBandMin(requiredLevel uint32) uint32 {
	if requiredLevel == 0 {
		return 1
	}
	return ((requiredLevel - 1) / EnhanceRequiredLevelBandSize * EnhanceRequiredLevelBandSize) + 1
}

// ResolveRequiredLevelBandMax 返回等级段终点（含）。
func ResolveRequiredLevelBandMax(bandMin uint32) uint32 {
	if bandMin == 0 {
		return EnhanceRequiredLevelBandSize
	}
	return bandMin + EnhanceRequiredLevelBandSize - 1
}

// FormatRequiredLevelBandLabel 生成后台/客户端可读的穿戴等级段文案。
func FormatRequiredLevelBandLabel(bandMin uint32) string {
	if bandMin == 0 {
		bandMin = 1
	}
	return fmt.Sprintf("%d~%d级", bandMin, ResolveRequiredLevelBandMax(bandMin))
}

// IsValidEnhanceRequiredLevelBandMin 校验后台提交的穿戴等级段起点。
func IsValidEnhanceRequiredLevelBandMin(bandMin uint32) bool {
	if bandMin == 0 || bandMin > MaxEnhanceRequiredLevelBandMin {
		return false
	}
	return (bandMin-1)%EnhanceRequiredLevelBandSize == 0
}

// ListEnhanceRequiredLevelBandMins 返回后台表单可选的穿戴等级段起点列表。
func ListEnhanceRequiredLevelBandMins() []uint32 {
	bands := make([]uint32, 0, 11)
	for bandMin := uint32(1); bandMin <= MaxEnhanceRequiredLevelBandMin; bandMin += EnhanceRequiredLevelBandSize {
		bands = append(bands, bandMin)
	}
	return bands
}
