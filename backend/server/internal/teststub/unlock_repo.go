package teststub

import (
	"context"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/unlock"
)

func NewUnlockRepository() *UnlockRepository {
	return &UnlockRepository{
		records: make(map[uint64]map[uint64]unlock.FeatureRecord),
	}
}

type UnlockRepository struct {
	mu      sync.RWMutex
	records map[uint64]map[uint64]unlock.FeatureRecord
}

func (r *UnlockRepository) GrantRuntimeFeature(_ context.Context, playerID uint64, featureID uint64, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*unlock.RuntimeGrantResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	playerRecords, ok := r.records[playerID]
	if !ok {
		playerRecords = make(map[uint64]unlock.FeatureRecord)
		r.records[playerID] = playerRecords
	}
	record, exists := playerRecords[featureID]
	if !exists {
		record = unlock.FeatureRecord{
			PlayerID:   playerID,
			FeatureID:  featureID,
			UnlockedAt: time.Now(),
		}
		playerRecords[featureID] = record
	}
	_ = reasonType
	_ = reasonRefID
	_ = operatorType
	_ = operatorID
	return &unlock.RuntimeGrantResult{
		Feature: record,
		Granted: !exists,
	}, nil
}
