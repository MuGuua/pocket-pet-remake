package postgres

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// unmarshalFlexibleUint32Array 兼容历史 JSON 脏数据：
// 技能列表既可能是 [101,102]，也可能是 ["101","102"]。
func unmarshalFlexibleUint32Array(data []byte) ([]uint32, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return []uint32{}, nil
	}

	var rawItems []any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&rawItems); err != nil {
		return nil, err
	}

	result := make([]uint32, 0, len(rawItems))
	for index, rawItem := range rawItems {
		value, err := parseFlexibleUint32(rawItem)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", index, err)
		}
		result = append(result, value)
	}
	return result, nil
}

// parseFlexibleUint32 把 JSON 数字或数字字符串统一转换成 uint32。
func parseFlexibleUint32(rawItem any) (uint32, error) {
	switch value := rawItem.(type) {
	case json.Number:
		parsed, err := strconv.ParseUint(value.String(), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid uint32 number %q: %w", value.String(), err)
		}
		return uint32(parsed), nil
	case string:
		trimmed := strings.TrimSpace(value)
		parsed, err := strconv.ParseUint(trimmed, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid uint32 string %q: %w", value, err)
		}
		return uint32(parsed), nil
	default:
		return 0, fmt.Errorf("unsupported value type %T", rawItem)
	}
}
