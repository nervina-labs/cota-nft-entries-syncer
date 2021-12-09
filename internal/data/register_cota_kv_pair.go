package data

import (
	"context"
	"github.com/nervina-labs/compact-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/compact-nft-entries-syncer/internal/logger"
	"gorm.io/gorm"
)

var _ biz.RegisterCotaKvPairRepo = (*registerCotaKvPairRepo)(nil)

type RegisterCotaKvPair struct {
	gorm.Model

	BlockNumber uint64
	LockHash    string
	LockHashCRC uint32
}

func NewRegisterCotaKvPairRepo(data *Data, logger *logger.Logger) *registerCotaKvPairRepo {
	return &registerCotaKvPairRepo{
		data:   data,
		logger: logger,
	}
}

type registerCotaKvPairRepo struct {
	data   *Data
	logger *logger.Logger
}

func (rp registerCotaKvPairRepo) CreateRegisterCotaKvPair(ctx context.Context, r *biz.RegisterCotaKvPair) error {
	if err := rp.data.db.WithContext(ctx).Create(r).Error; err != nil {
		return err
	}
	return nil
}

func (rp registerCotaKvPairRepo) DeleteRegisterCotaKvPairs(ctx context.Context, blockNumber uint64) error {
	if err := rp.data.db.WithContext(ctx).Where("block_number = ?", blockNumber).Delete(RegisterCotaKvPair{}).Error; err != nil {
		return err
	}
	return nil
}
