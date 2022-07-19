package biz

import (
	"context"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
)

type WithdrawQueryInfo struct {
	BlockNumber uint64
	OutPoint    string
	LockHash    string
}

type WithdrawExtraInfoRepo interface {
	CreateExtraInfo(ctx context.Context, outPoint string, txHash string, lockScriptId uint) error
	FindQueryInfos(ctx context.Context, page int, pageSize int) ([]WithdrawQueryInfo, error)
	FindOrCreateScript(ctx context.Context, script *Script) error
}

type WithdrawExtraInfoUsecase struct {
	repo   WithdrawExtraInfoRepo
	logger *logger.Logger
}

func NewWithdrawExtraInfoUsecase(repo WithdrawExtraInfoRepo, logger *logger.Logger) *WithdrawExtraInfoUsecase {
	return &WithdrawExtraInfoUsecase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *WithdrawExtraInfoUsecase) CreateExtraInfo(ctx context.Context, outPoint string, txHash string, lockScriptId uint) error {
	return uc.repo.CreateExtraInfo(ctx, outPoint, txHash, lockScriptId)
}

func (uc *WithdrawExtraInfoUsecase) FindQueryInfos(ctx context.Context, page int, pageSize int) ([]WithdrawQueryInfo, error) {
	return uc.repo.FindQueryInfos(ctx, page, pageSize)
}

func (uc *WithdrawExtraInfoUsecase) FindOrCreateScript(ctx context.Context, script *Script) error {
	return uc.repo.FindOrCreateScript(ctx, script)
}
