package unlock

import "context"

// Repository 定义运行时功能解锁所需的最小持久化能力。
// 当前先只提供授予能力，后续如果客户端需要拉取全量解锁列表，再继续扩展查询接口。
type Repository interface {
	GrantRuntimeFeature(ctx context.Context, playerID uint64, featureID uint64, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*RuntimeGrantResult, error)
}
