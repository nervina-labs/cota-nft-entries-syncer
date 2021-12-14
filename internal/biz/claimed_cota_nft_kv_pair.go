package biz

import (
	"context"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
)

type ClaimedCotaNftKvPair struct {
	BlockNumber uint64
	CotaId      string
	CotaIdCRC   uint32
	Total       uint32
	TokenIndex  uint32
	OutPoint    string
	OutPointCrc uint32
	LockHash    string
	LockHashCrc uint32
}

type ClaimedCotaNftKvPairRepo interface {
	CreateClaimedCotaNftKvPair(ctx context.Context, w *ClaimedCotaNftKvPair) error
	DeleteClaimedCotaNftKvPairs(ctx context.Context, blockNumber uint64) error
}

type ClaimedCotaNftKvPairUsecase struct {
	repo   ClaimedCotaNftKvPairRepo
	logger *logger.Logger
}

func NewClaimedCotaNftKvPairUsecase(repo ClaimedCotaNftKvPairRepo, logger *logger.Logger) *ClaimedCotaNftKvPairUsecase {
	return &ClaimedCotaNftKvPairUsecase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *ClaimedCotaNftKvPairUsecase) Create(ctx context.Context, c *ClaimedCotaNftKvPair) error {
	return uc.repo.CreateClaimedCotaNftKvPair(ctx, c)
}

func (uc *ClaimedCotaNftKvPairUsecase) DeleteByBlockNumber(ctx context.Context, blockNumber uint64) error {
	return uc.repo.DeleteClaimedCotaNftKvPairs(ctx, blockNumber)
}
