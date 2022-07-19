package service

import (
	"context"
	"encoding/hex"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/data"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
	ckbTypes "github.com/nervosnetwork/ckb-sdk-go/types"
)

const pageSize int = 10000

var _ Service = (*WithdrawExtraInfoService)(nil)

type WithdrawExtraInfoService struct {
	extraInfoUsecase *biz.WithdrawExtraInfoUsecase
	logger           *logger.Logger
	client           *data.CkbNodeClient
}

func NewWithdrawExtraInfoService(extraInfoUsecase *biz.WithdrawExtraInfoUsecase, logger *logger.Logger, client *data.CkbNodeClient) *WithdrawExtraInfoService {
	return &WithdrawExtraInfoService{
		extraInfoUsecase: extraInfoUsecase,
		logger:           logger,
		client:           client,
	}
}

func (s WithdrawExtraInfoService) Start(ctx context.Context, _ string) error {
	s.logger.Info(ctx, "withdraw extra info service started")
	page := 0
	for {
		queryInfos, err := s.extraInfoUsecase.FindQueryInfos(ctx, page, pageSize)
		if err != nil {
			return err
		}
		if len(queryInfos) == 0 {
			break
		}
		page += 1

		if err = s.parseExtraInfos(ctx, queryInfos); err != nil {
			return err
		}
	}
	return nil
}

func (s WithdrawExtraInfoService) Stop(ctx context.Context) error {
	s.logger.Info(ctx, "withdraw extra info service stopped")

	return nil
}

func (s WithdrawExtraInfoService) parseExtraInfos(ctx context.Context, infos []biz.WithdrawQueryInfo) (err error) {
	var block *ckbTypes.Block
	for _, info := range infos {
		block, err = s.client.Rpc.GetBlockByNumber(ctx, info.BlockNumber)
		if err != nil {
			return
		}
		for _, tx := range block.Transactions {
			if err = s.parseExtraInfo(ctx, tx, info); err != nil {
				return
			}
		}
	}
	return
}

func (s WithdrawExtraInfoService) parseExtraInfo(ctx context.Context, tx *ckbTypes.Transaction, info biz.WithdrawQueryInfo) error {
	for _, input := range tx.Inputs {
		// Find CoTA transactions with txHash and outPoint
		if hex.EncodeToString(input.PreviousOutput.TxHash[12:]) != info.OutPoint[:40] {
			continue
		}
		for _, output := range tx.Outputs {
			lockHash, err := output.Lock.Hash()
			if err != nil {
				return err
			}
			if lockHash.String()[2:] != info.LockHash {
				continue
			}
			hashType, err := output.Lock.HashType.Serialize()
			if err != nil {
				return err
			}
			lock := biz.Script{
				CodeHash: hex.EncodeToString(output.Lock.CodeHash[:]),
				HashType: hex.EncodeToString(hashType),
				Args:     hex.EncodeToString(output.Lock.Args),
			}
			if err = s.extraInfoUsecase.FindOrCreateScript(ctx, &lock); err != nil {
				return err
			}
			if err = s.extraInfoUsecase.CreateExtraInfo(ctx, info.OutPoint, tx.Hash.String()[2:], lock.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
