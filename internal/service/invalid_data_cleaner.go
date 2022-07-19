package service

import (
	"context"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/data"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
)

var _ Service = (*InvalidDataCleaner)(nil)

type InvalidDataCleaner struct {
	invalidDataUsecase *biz.InvalidDataUsecase
	logger             *logger.Logger
	client             *data.CkbNodeClient
}

func NewInvalidDataService(invalidDataUsecase *biz.InvalidDataUsecase, logger *logger.Logger, client *data.CkbNodeClient) *InvalidDataCleaner {
	return &InvalidDataCleaner{
		invalidDataUsecase: invalidDataUsecase,
		logger:             logger,
		client:             client,
	}
}

func (i InvalidDataCleaner) Start(ctx context.Context, _ string) error {
	var blockNumber uint64
	info, err := i.client.Rpc.GetBlockchainInfo(ctx)
	if err != nil {
		return err
	}

	// The block number where the cota smart contract was deployed
	if info.Chain == "ckb" {
		blockNumber = 7233113
	} else {
		blockNumber = 5476282
	}

	return i.invalidDataUsecase.Clean(ctx, blockNumber)
}

func (i InvalidDataCleaner) Stop(ctx context.Context) error {
	i.logger.Info(ctx, "invalid data cleaner stopped")

	return nil
}
