package data

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"
	"time"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
	"github.com/nervina-labs/cota-smt-go/smt"
)

var _ biz.DefineCotaNftKvPairRepo = (*defineCotaNftKvPairRepo)(nil)

type DefineCotaNftKvPair struct {
	ID          uint `gorm:"primaryKey"`
	BlockNumber uint64
	CotaId      string
	Total       uint32
	Issued      uint32
	Configure   uint8
	LockHash    string
	LockHashCRC uint32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DefineCotaNftKvPairVersion struct {
	ID             uint `gorm:"primaryKey"`
	OldBlockNumber uint64
	BlockNumber    uint64
	CotaId         string
	Total          uint32
	OldIssued      uint32
	Issued         uint32
	Configure      uint8
	LockHash       string
	ActionType     uint8 //	0-create 1-update 2-delete
	TxIndex        uint32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type defineCotaNftKvPairRepo struct {
	data   *Data
	logger *logger.Logger
}

func NewDefineCotaNftKvPairRepo(data *Data, logger *logger.Logger) biz.DefineCotaNftKvPairRepo {
	return &defineCotaNftKvPairRepo{
		data:   data,
		logger: logger,
	}
}

func (rp defineCotaNftKvPairRepo) CreateDefineCotaNftKvPair(ctx context.Context, d *biz.DefineCotaNftKvPair) error {
	if err := rp.data.db.WithContext(ctx).Create(d).Error; err != nil {
		return err
	}
	return nil
}

func (rp defineCotaNftKvPairRepo) DeleteDefineCotaNftKvPairs(ctx context.Context, blockNumber uint64) error {
	if err := rp.data.db.WithContext(ctx).Where("block_number = ?", blockNumber).Error; err != nil {
		return err
	}
	return nil
}

func (rp defineCotaNftKvPairRepo) ParseDefineCotaEntries(blockNumber uint64, entry biz.Entry) (defineCotas []biz.DefineCotaNftKvPair, err error) {
	entries := smt.DefineCotaNFTEntriesFromSliceUnchecked(entry.InputType[1:])
	defineCotaKeyVec := entries.DefineKeys()
	defineCotaValueVec := entries.DefineValues()
	lockHash, err := entry.LockScript.Hash()
	if err != nil {
		return
	}
	lockHashStr := lockHash.String()[2:]
	lockHashCRC32 := crc32.ChecksumIEEE([]byte(lockHashStr))
	for i := uint(0); i < defineCotaKeyVec.Len(); i++ {
		key := defineCotaKeyVec.Get(i)
		value := defineCotaValueVec.Get(i)
		defineCotas = append(defineCotas, biz.DefineCotaNftKvPair{
			BlockNumber: blockNumber,
			CotaId:      hex.EncodeToString(key.CotaId().RawData()),
			Total:       binary.BigEndian.Uint32(value.Total().RawData()),
			Issued:      binary.BigEndian.Uint32(value.Issued().RawData()),
			Configure:   value.Configure().AsSlice()[0],
			LockHash:    lockHashStr,
			LockHashCRC: lockHashCRC32,
			TxIndex:     entry.TxIndex,
		})
	}
	return
}
