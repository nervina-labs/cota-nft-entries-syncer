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

var _ biz.HoldCotaNftKvPairRepo = (*holdCotaNftKvPairRepo)(nil)

type HoldCotaNftKvPair struct {
	ID             uint `gorm:"primaryKey"`
	BlockNumber    uint64
	CotaId         string
	TokenIndex     uint32
	State          uint8
	Configure      uint8
	Characteristic string
	LockHash       string
	LockHashCRC    uint32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type HoldCotaNftKvPairVersion struct {
	ID                uint `gorm:"primaryKey"`
	OldBlockNumber    uint64
	BlockNumber       uint64
	CotaId            string
	TokenIndex        uint32
	OldState          uint8
	State             uint8
	Configure         uint8
	OldCharacteristic string
	Characteristic    string
	OldLockHash       string
	LockHash          string
	ActionType        uint8 //	0-create 1-update 2-delete
	TxIndex           uint32
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type holdCotaNftKvPairRepo struct {
	data   *Data
	logger *logger.Logger
}

func NewHoldCotaNftKvPairRepo(data *Data, logger *logger.Logger) biz.HoldCotaNftKvPairRepo {
	return &holdCotaNftKvPairRepo{
		data:   data,
		logger: logger,
	}
}

func (rp holdCotaNftKvPairRepo) CreateHoldCotaNftKvPair(ctx context.Context, h *biz.HoldCotaNftKvPair) error {
	if err := rp.data.db.WithContext(ctx).Create(h).Error; err != nil {
		return err
	}
	return nil
}

func (rp holdCotaNftKvPairRepo) DeleteHoldCotaNftKvPairs(ctx context.Context, blockNumber uint64) error {
	if err := rp.data.db.WithContext(ctx).Where("block_number = ?", blockNumber).Delete(HoldCotaNftKvPair{}).Error; err != nil {
		return err
	}
	return nil
}

func (rp holdCotaNftKvPairRepo) ParseHoldCotaEntries(blockNumber uint64, entry biz.Entry) (holdCotas []biz.HoldCotaNftKvPair, err error) {
	entries := smt.UpdateCotaNFTEntriesFromSliceUnchecked(entry.InputType[1:])
	holdCotaKeyVec := entries.HoldKeys()
	holdCotaValueVec := entries.HoldNewValues()
	lockHash, err := entry.LockScript.Hash()
	if err != nil {
		return
	}
	lockHashStr := lockHash.String()[2:]
	lockHashCRC32 := crc32.ChecksumIEEE([]byte(lockHashStr))
	for i := uint(0); i < holdCotaKeyVec.Len(); i++ {
		key := holdCotaKeyVec.Get(i)
		value := holdCotaValueVec.Get(i)
		holdCotas = append(holdCotas, biz.HoldCotaNftKvPair{
			BlockNumber:    blockNumber,
			CotaId:         hex.EncodeToString(key.CotaId().RawData()),
			TokenIndex:     binary.BigEndian.Uint32(key.Index().RawData()),
			State:          value.State().AsSlice()[0],
			Configure:      value.Configure().AsSlice()[0],
			Characteristic: hex.EncodeToString(value.Characteristic().RawData()),
			LockHash:       lockHashStr,
			LockHashCRC:    lockHashCRC32,
			TxIndex:        entry.TxIndex,
			UpdatedAt:      time.Now().UTC(),
		})
	}
	return
}
