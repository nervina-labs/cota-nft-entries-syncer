package data

import (
	"context"
	"hash/crc32"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
)

var _ biz.WithdrawExtraInfoRepo = (*withdrawExtraInfoRepo)(nil)

type withdrawExtraInfoRepo struct {
	data   *Data
	logger *logger.Logger
}

func NewWithdrawExtraInfoRepo(data *Data, logger *logger.Logger) biz.WithdrawExtraInfoRepo {
	return &withdrawExtraInfoRepo{
		data:   data,
		logger: logger,
	}
}

func (rp withdrawExtraInfoRepo) CreateExtraInfo(ctx context.Context, outPoint string, txHash string, lockScriptId uint) error {
	outPointCrc := crc32.ChecksumIEEE([]byte(outPoint))
	if err := rp.data.db.WithContext(ctx).Model(WithdrawCotaNftKvPair{}).Where("out_point_crc = ? AND out_point = ?", outPointCrc, outPoint).Updates(WithdrawCotaNftKvPair{TxHash: txHash, LockScriptId: lockScriptId}).Error; err != nil {
		return err
	}
	return nil
}

func (rp withdrawExtraInfoRepo) FindQueryInfos(ctx context.Context, page int, pageSize int) ([]biz.WithdrawQueryInfo, error) {
	var (
		withdrawals, temps []WithdrawCotaNftKvPair
		queryInfos         []biz.WithdrawQueryInfo
		offset             int
	)
	for {
		result := rp.data.db.WithContext(ctx).Select("DISTINCT out_point, block_number, lock_hash").Where("tx_hash = ''").Limit(pageSize).Offset(offset * pageSize).Find(&temps)
		if result.Error != nil {
			return queryInfos, result.Error
		}
		withdrawals = append(withdrawals, temps...)
		if result.RowsAffected == 0 {
			break
		}
		offset++
	}
	for _, v := range withdrawals {
		queryInfos = append(queryInfos, biz.WithdrawQueryInfo{
			BlockNumber: v.BlockNumber,
			OutPoint:    v.OutPoint,
			LockHash:    v.LockHash,
		})
	}
	return queryInfos, nil
}

func (rp withdrawExtraInfoRepo) FindOrCreateScript(ctx context.Context, script *biz.Script) error {
	ht, err := hashType(script.HashType)
	if err != nil {
		return err
	}
	s := Script{}
	if err = rp.data.db.WithContext(ctx).FirstOrCreate(&s, Script{
		CodeHash:    script.CodeHash,
		CodeHashCrc: crc32.ChecksumIEEE([]byte(script.CodeHash)),
		HashType:    ht,
		Args:        script.Args,
		ArgsCrc:     crc32.ChecksumIEEE([]byte(script.Args)),
	}).Error; err != nil {
		return err
	}
	script.ID = s.ID
	return nil
}
