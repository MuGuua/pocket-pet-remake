package pet

import (
	"math/rand"
	"strings"
)

const (
	// AcquireMethodWildCapture 标记可通过野外捕捉获得的系统宠物模板。
	AcquireMethodWildCapture = "wild_capture"
	// GrantSourceTemplate 表示按模板固定资质发放。
	GrantSourceTemplate = "template"
	// GrantSourceWildCapture 表示野外捕捉成功后 roll 资质发放。
	GrantSourceWildCapture = "wild_capture"
)

// GrowthAptitudes 描述宠物实例的五项成长资质。
type GrowthAptitudes struct {
	HPApt   uint32
	ATKApt  uint32
	DEFApt  uint32
	SPDApt  uint32
	MANAApt uint32
}

// AptitudeRollRanges 描述野外捕捉模板每项资质的 roll 区间。
type AptitudeRollRanges struct {
	HPAptMin   uint32
	HPAptMax   uint32
	ATKAptMin  uint32
	ATKAptMax  uint32
	DEFAptMin  uint32
	DEFAptMax  uint32
	SPDAptMin  uint32
	SPDAptMax  uint32
	MANAAptMin uint32
	MANAAptMax uint32
}

// IsWildCaptureAcquireMethod 判断模板获取方式是否属于野外捕捉类。
func IsWildCaptureAcquireMethod(acquireMethod string) bool {
	normalized := strings.TrimSpace(strings.ToLower(acquireMethod))
	return normalized == AcquireMethodWildCapture || strings.Contains(acquireMethod, "野外捕捉")
}

// ValidateAptitudeRollRanges 校验野外捕捉模板的资质 roll 区间是否合法。
func ValidateAptitudeRollRanges(ranges AptitudeRollRanges) error {
	checks := []struct {
		min uint32
		max uint32
	}{
		{ranges.HPAptMin, ranges.HPAptMax},
		{ranges.ATKAptMin, ranges.ATKAptMax},
		{ranges.DEFAptMin, ranges.DEFAptMax},
		{ranges.SPDAptMin, ranges.SPDAptMax},
		{ranges.MANAAptMin, ranges.MANAAptMax},
	}
	for _, item := range checks {
		if item.min == 0 || item.max == 0 {
			return ErrInvalidAptitudeRollRange
		}
		if item.min > item.max {
			return ErrInvalidAptitudeRollRange
		}
	}
	return nil
}

// RollWildCaptureAptitudes 在模板配置的区间内随机生成五项资质。
func RollWildCaptureAptitudes(ranges AptitudeRollRanges, rng *rand.Rand) GrowthAptitudes {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	return GrowthAptitudes{
		HPApt:   rollUint32Inclusive(ranges.HPAptMin, ranges.HPAptMax, rng),
		ATKApt:  rollUint32Inclusive(ranges.ATKAptMin, ranges.ATKAptMax, rng),
		DEFApt:  rollUint32Inclusive(ranges.DEFAptMin, ranges.DEFAptMax, rng),
		SPDApt:  rollUint32Inclusive(ranges.SPDAptMin, ranges.SPDAptMax, rng),
		MANAApt: rollUint32Inclusive(ranges.MANAAptMin, ranges.MANAAptMax, rng),
	}
}

func rollUint32Inclusive(minValue uint32, maxValue uint32, rng *rand.Rand) uint32 {
	if minValue >= maxValue {
		return minValue
	}
	return minValue + uint32(rng.Intn(int(maxValue-minValue+1)))
}
