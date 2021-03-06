package data

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/data/blockchain"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
	"github.com/nervina-labs/cota-smt-go/smt"
)

var _ biz.TransferCotaKvPairRepo = (*transferCotaKvPairRepo)(nil)

type transferCotaKvPairRepo struct {
	data   *Data
	logger *logger.Logger
}

func (rp transferCotaKvPairRepo) ParseTransferCotaEntries(blockNumber uint64, entry biz.Entry) (claimedCotas []biz.ClaimedCotaNftKvPair, withdrawCotas []biz.WithdrawCotaNftKvPair, err error) {
	if entry.Version == 0 {
		return generateTransferWithdrawV0KvPairs(blockNumber, entry, rp)
	}
	return generateTransferWithdrawV1ToV2KvPairs(blockNumber, entry, rp)
}

func (rp transferCotaKvPairRepo) ParseTransferUpdateCotaEntries(blockNumber uint64, entry biz.Entry) (claimedCotas []biz.ClaimedCotaNftKvPair, withdrawCotas []biz.WithdrawCotaNftKvPair, err error) {
	if entry.Version == 0 {
		return generateTransferUpdateWithdrawV0KvPairs(blockNumber, entry, rp)
	}
	return generateTransferUpdateWithdrawV1ToV2KvPairs(blockNumber, entry, rp)
}

func (rp transferCotaKvPairRepo) FindOrCreateScript(ctx context.Context, script *biz.Script) error {
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

func NewTransferCotaKvPairRepo(data *Data, logger *logger.Logger) biz.TransferCotaKvPairRepo {
	return &transferCotaKvPairRepo{
		data:   data,
		logger: logger,
	}
}

func generateTransferUpdateWithdrawV0KvPairs(blockNumber uint64, entry biz.Entry, rp transferCotaKvPairRepo) (claimedCotas []biz.ClaimedCotaNftKvPair, withdrawCotas []biz.WithdrawCotaNftKvPair, err error) {
	entries := smt.TransferUpdateCotaNFTEntriesFromSliceUnchecked(entry.InputType[1:])
	claimedCotaKeyVec := entries.ClaimKeys()
	lockHash, err := entry.LockScript.Hash()
	if err != nil {
		return
	}
	lockHashStr := lockHash.String()[2:]
	lockHashCRC32 := crc32.ChecksumIEEE([]byte(lockHashStr))
	for i := uint(0); i < claimedCotaKeyVec.Len(); i++ {
		key := claimedCotaKeyVec.Get(i)
		cotaId := hex.EncodeToString(key.NftId().CotaId().RawData())
		outpointStr := hex.EncodeToString(key.OutPoint().RawData())
		claimedCotas = append(claimedCotas, biz.ClaimedCotaNftKvPair{
			BlockNumber: blockNumber,
			CotaId:      hex.EncodeToString(key.NftId().CotaId().RawData()),
			CotaIdCRC:   crc32.ChecksumIEEE([]byte(cotaId)),
			TokenIndex:  binary.BigEndian.Uint32(key.NftId().Index().RawData()),
			OutPoint:    outpointStr,
			OutPointCrc: crc32.ChecksumIEEE([]byte(outpointStr)),
			LockHash:    lockHashStr,
			LockHashCrc: lockHashCRC32,
		})
	}
	withdrawKeyVec := entries.WithdrawalKeys()
	withdrawValueVec := entries.WithdrawalValues()
	for i := uint(0); i < withdrawKeyVec.Len(); i++ {
		key := withdrawKeyVec.Get(i)
		value := withdrawValueVec.Get(i)
		cotaId := hex.EncodeToString(key.CotaId().RawData())
		outpointStr := hex.EncodeToString(value.OutPoint().RawData())
		receiverLock := blockchain.ScriptFromSliceUnchecked(value.ToLock().RawData())
		script := biz.Script{
			CodeHash: hex.EncodeToString(receiverLock.CodeHash().RawData()),
			HashType: hex.EncodeToString(receiverLock.HashType().AsSlice()),
			Args:     hex.EncodeToString(receiverLock.Args().RawData()),
		}
		err = rp.FindOrCreateScript(context.TODO(), &script)
		if err != nil {
			return
		}
		withdrawCotas = append(withdrawCotas, biz.WithdrawCotaNftKvPair{
			BlockNumber:          blockNumber,
			CotaId:               cotaId,
			CotaIdCRC:            crc32.ChecksumIEEE([]byte(cotaId)),
			TokenIndex:           binary.BigEndian.Uint32(key.Index().RawData()),
			OutPoint:             outpointStr,
			OutPointCrc:          crc32.ChecksumIEEE([]byte(outpointStr)),
			State:                value.NftInfo().State().AsSlice()[0],
			Configure:            value.NftInfo().Configure().AsSlice()[0],
			Characteristic:       hex.EncodeToString(value.NftInfo().Characteristic().RawData()),
			ReceiverLockScriptId: script.ID,
			LockHash:             lockHashStr,
			LockHashCrc:          lockHashCRC32,
			Version:              entry.Version,
		})
	}
	return
}

func generateTransferUpdateWithdrawV1ToV2KvPairs(blockNumber uint64, entry biz.Entry, rp transferCotaKvPairRepo) (claimedCotas []biz.ClaimedCotaNftKvPair, withdrawCotas []biz.WithdrawCotaNftKvPair, err error) {
	var claimedCotaKeyVec *smt.ClaimCotaNFTKeyVec = nil
	var withdrawKeyVec *smt.WithdrawalCotaNFTKeyV1Vec = nil
	var withdrawValueVec *smt.WithdrawalCotaNFTValueV1Vec = nil

	if entry.Version == 1 {
		entries := smt.TransferUpdateCotaNFTV1EntriesFromSliceUnchecked(entry.InputType[1:])
		claimedCotaKeyVec = entries.ClaimKeys()
		withdrawKeyVec = entries.WithdrawalKeys()
		withdrawValueVec = entries.WithdrawalValues()
	} else {
		entries := smt.TransferUpdateCotaNFTV2EntriesFromSliceUnchecked(entry.InputType[1:])
		claimedCotaKeyVec = entries.ClaimKeys()
		withdrawKeyVec = entries.WithdrawalKeys()
		withdrawValueVec = entries.WithdrawalValues()
	}
	lockHash, err := entry.LockScript.Hash()
	if err != nil {
		return
	}
	lockHashStr := lockHash.String()[2:]
	lockHashCRC32 := crc32.ChecksumIEEE([]byte(lockHashStr))
	for i := uint(0); i < claimedCotaKeyVec.Len(); i++ {
		key := claimedCotaKeyVec.Get(i)
		cotaId := hex.EncodeToString(key.NftId().CotaId().RawData())
		outpointStr := hex.EncodeToString(key.OutPoint().RawData())
		claimedCotas = append(claimedCotas, biz.ClaimedCotaNftKvPair{
			BlockNumber: blockNumber,
			CotaId:      hex.EncodeToString(key.NftId().CotaId().RawData()),
			CotaIdCRC:   crc32.ChecksumIEEE([]byte(cotaId)),
			TokenIndex:  binary.BigEndian.Uint32(key.NftId().Index().RawData()),
			OutPoint:    outpointStr,
			OutPointCrc: crc32.ChecksumIEEE([]byte(outpointStr)),
			LockHash:    lockHashStr,
			LockHashCrc: lockHashCRC32,
		})
	}
	for i := uint(0); i < withdrawKeyVec.Len(); i++ {
		key := withdrawKeyVec.Get(i)
		value := withdrawValueVec.Get(i)
		cotaId := hex.EncodeToString(key.NftId().CotaId().RawData())
		outpointStr := hex.EncodeToString(key.OutPoint().RawData())
		receiverLock := blockchain.ScriptFromSliceUnchecked(value.ToLock().RawData())
		script := biz.Script{
			CodeHash: hex.EncodeToString(receiverLock.CodeHash().RawData()),
			HashType: hex.EncodeToString(receiverLock.HashType().AsSlice()),
			Args:     hex.EncodeToString(receiverLock.Args().RawData()),
		}
		err = rp.FindOrCreateScript(context.TODO(), &script)
		if err != nil {
			return
		}
		withdrawCotas = append(withdrawCotas, biz.WithdrawCotaNftKvPair{
			BlockNumber:          blockNumber,
			CotaId:               cotaId,
			CotaIdCRC:            crc32.ChecksumIEEE([]byte(cotaId)),
			TokenIndex:           binary.BigEndian.Uint32(key.NftId().Index().RawData()),
			OutPoint:             outpointStr,
			OutPointCrc:          crc32.ChecksumIEEE([]byte(outpointStr)),
			State:                value.NftInfo().State().AsSlice()[0],
			Configure:            value.NftInfo().Configure().AsSlice()[0],
			Characteristic:       hex.EncodeToString(value.NftInfo().Characteristic().RawData()),
			ReceiverLockScriptId: script.ID,
			LockHash:             lockHashStr,
			LockHashCrc:          lockHashCRC32,
			Version:              entry.Version,
		})
	}
	return
}

func generateTransferWithdrawV0KvPairs(blockNumber uint64, entry biz.Entry, rp transferCotaKvPairRepo) (claimedCotas []biz.ClaimedCotaNftKvPair, withdrawCotas []biz.WithdrawCotaNftKvPair, err error) {
	entries := smt.TransferCotaNFTEntriesFromSliceUnchecked(entry.InputType[1:])
	claimedCotaKeyVec := entries.ClaimKeys()
	lockHash, err := entry.LockScript.Hash()
	if err != nil {
		return
	}
	lockHashStr := lockHash.String()[2:]
	lockHashCRC32 := crc32.ChecksumIEEE([]byte(lockHashStr))
	for i := uint(0); i < claimedCotaKeyVec.Len(); i++ {
		key := claimedCotaKeyVec.Get(i)
		cotaId := hex.EncodeToString(key.NftId().CotaId().RawData())
		outpointStr := hex.EncodeToString(key.OutPoint().RawData())
		claimedCotas = append(claimedCotas, biz.ClaimedCotaNftKvPair{
			BlockNumber: blockNumber,
			CotaId:      hex.EncodeToString(key.NftId().CotaId().RawData()),
			CotaIdCRC:   crc32.ChecksumIEEE([]byte(cotaId)),
			TokenIndex:  binary.BigEndian.Uint32(key.NftId().Index().RawData()),
			OutPoint:    outpointStr,
			OutPointCrc: crc32.ChecksumIEEE([]byte(outpointStr)),
			LockHash:    lockHashStr,
			LockHashCrc: lockHashCRC32,
		})
	}
	withdrawKeyVec := entries.WithdrawalKeys()
	withdrawValueVec := entries.WithdrawalValues()
	for i := uint(0); i < withdrawKeyVec.Len(); i++ {
		key := withdrawKeyVec.Get(i)
		value := withdrawValueVec.Get(i)
		cotaId := hex.EncodeToString(key.CotaId().RawData())
		outpointStr := hex.EncodeToString(value.OutPoint().RawData())
		receiverLock := blockchain.ScriptFromSliceUnchecked(value.ToLock().RawData())
		script := biz.Script{
			CodeHash: hex.EncodeToString(receiverLock.CodeHash().RawData()),
			HashType: hex.EncodeToString(receiverLock.HashType().AsSlice()),
			Args:     hex.EncodeToString(receiverLock.Args().RawData()),
		}
		err = rp.FindOrCreateScript(context.TODO(), &script)
		if err != nil {
			return
		}
		withdrawCotas = append(withdrawCotas, biz.WithdrawCotaNftKvPair{
			BlockNumber:          blockNumber,
			CotaId:               cotaId,
			CotaIdCRC:            crc32.ChecksumIEEE([]byte(cotaId)),
			TokenIndex:           binary.BigEndian.Uint32(key.Index().RawData()),
			OutPoint:             outpointStr,
			OutPointCrc:          crc32.ChecksumIEEE([]byte(outpointStr)),
			State:                value.NftInfo().State().AsSlice()[0],
			Configure:            value.NftInfo().Configure().AsSlice()[0],
			Characteristic:       hex.EncodeToString(value.NftInfo().Characteristic().RawData()),
			ReceiverLockScriptId: script.ID,
			LockHash:             lockHashStr,
			LockHashCrc:          lockHashCRC32,
			Version:              entry.Version,
		})
	}
	return
}

func generateTransferWithdrawV1ToV2KvPairs(blockNumber uint64, entry biz.Entry, rp transferCotaKvPairRepo) (claimedCotas []biz.ClaimedCotaNftKvPair, withdrawCotas []biz.WithdrawCotaNftKvPair, err error) {
	var claimedCotaKeyVec *smt.ClaimCotaNFTKeyVec = nil
	var withdrawKeyVec *smt.WithdrawalCotaNFTKeyV1Vec = nil
	var withdrawValueVec *smt.WithdrawalCotaNFTValueV1Vec = nil
	if entry.Version == 1 {
		entries := smt.TransferCotaNFTV1EntriesFromSliceUnchecked(entry.InputType[1:])
		claimedCotaKeyVec = entries.ClaimKeys()
		withdrawKeyVec = entries.WithdrawalKeys()
		withdrawValueVec = entries.WithdrawalValues()
	} else {
		entries := smt.TransferCotaNFTV2EntriesFromSliceUnchecked(entry.InputType[1:])
		claimedCotaKeyVec = entries.ClaimKeys()
		withdrawKeyVec = entries.WithdrawalKeys()
		withdrawValueVec = entries.WithdrawalValues()
	}
	lockHash, err := entry.LockScript.Hash()
	if err != nil {
		return
	}
	lockHashStr := lockHash.String()[2:]
	lockHashCRC32 := crc32.ChecksumIEEE([]byte(lockHashStr))
	for i := uint(0); i < claimedCotaKeyVec.Len(); i++ {
		key := claimedCotaKeyVec.Get(i)
		cotaId := hex.EncodeToString(key.NftId().CotaId().RawData())
		outpointStr := hex.EncodeToString(key.OutPoint().RawData())
		claimedCotas = append(claimedCotas, biz.ClaimedCotaNftKvPair{
			BlockNumber: blockNumber,
			CotaId:      hex.EncodeToString(key.NftId().CotaId().RawData()),
			CotaIdCRC:   crc32.ChecksumIEEE([]byte(cotaId)),
			TokenIndex:  binary.BigEndian.Uint32(key.NftId().Index().RawData()),
			OutPoint:    outpointStr,
			OutPointCrc: crc32.ChecksumIEEE([]byte(outpointStr)),
			LockHash:    lockHashStr,
			LockHashCrc: lockHashCRC32,
		})
	}
	for i := uint(0); i < withdrawKeyVec.Len(); i++ {
		key := withdrawKeyVec.Get(i)
		value := withdrawValueVec.Get(i)
		cotaId := hex.EncodeToString(key.NftId().CotaId().RawData())
		outpointStr := hex.EncodeToString(key.OutPoint().RawData())
		receiverLock := blockchain.ScriptFromSliceUnchecked(value.ToLock().RawData())
		script := biz.Script{
			CodeHash: hex.EncodeToString(receiverLock.CodeHash().RawData()),
			HashType: hex.EncodeToString(receiverLock.HashType().AsSlice()),
			Args:     hex.EncodeToString(receiverLock.Args().RawData()),
		}
		err = rp.FindOrCreateScript(context.TODO(), &script)
		if err != nil {
			return
		}
		withdrawCotas = append(withdrawCotas, biz.WithdrawCotaNftKvPair{
			BlockNumber:          blockNumber,
			CotaId:               cotaId,
			CotaIdCRC:            crc32.ChecksumIEEE([]byte(cotaId)),
			TokenIndex:           binary.BigEndian.Uint32(key.NftId().Index().RawData()),
			OutPoint:             outpointStr,
			OutPointCrc:          crc32.ChecksumIEEE([]byte(outpointStr)),
			State:                value.NftInfo().State().AsSlice()[0],
			Configure:            value.NftInfo().Configure().AsSlice()[0],
			Characteristic:       hex.EncodeToString(value.NftInfo().Characteristic().RawData()),
			ReceiverLockScriptId: script.ID,
			LockHash:             lockHashStr,
			LockHashCrc:          lockHashCRC32,
			Version:              entry.Version,
		})
	}
	return
}
