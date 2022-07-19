package data

import (
	"context"
	"errors"
	"hash/crc32"
	"time"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ biz.KvPairRepo = (*kvPairRepo)(nil)

type kvPairRepo struct {
	data   *Data
	logger *logger.Logger
}

func NewKvPairRepo(data *Data, logger *logger.Logger) biz.KvPairRepo {
	return &kvPairRepo{
		data:   data,
		logger: logger,
	}
}

func (rp kvPairRepo) CreateCotaEntryKvPairs(ctx context.Context, checkInfo biz.CheckInfo, kvPair *biz.KvPair) error {
	return rp.data.db.Transaction(func(tx *gorm.DB) error {
		// create register cotas
		if kvPair.HasRegisters() {
			registers := make([]RegisterCotaKvPair, len(kvPair.Registers))
			for i, register := range kvPair.Registers {
				registers[i] = RegisterCotaKvPair{
					BlockNumber: register.BlockNumber,
					LockHash:    register.LockHash,
				}
			}
			if err := tx.Model(RegisterCotaKvPair{}).WithContext(ctx).Create(registers).Error; err != nil {
				return err
			}
		}
		// create define cotas
		if kvPair.HasDefineCotas() {
			defineCotas := make([]DefineCotaNftKvPair, len(kvPair.DefineCotas))
			for i, cota := range kvPair.DefineCotas {
				defineCotas[i] = DefineCotaNftKvPair{
					BlockNumber: cota.BlockNumber,
					CotaId:      cota.CotaId,
					Total:       cota.Total,
					Issued:      cota.Issued,
					Configure:   cota.Configure,
					LockHash:    cota.LockHash,
					LockHashCRC: cota.LockHashCRC,
				}
			}
			if err := tx.Debug().Model(DefineCotaNftKvPair{}).WithContext(ctx).Create(defineCotas).Error; err != nil {
				return err
			}
			defineCotaVersions := make([]DefineCotaNftKvPairVersion, len(kvPair.DefineCotas))
			for i, define := range kvPair.DefineCotas {
				defineCotaVersion := DefineCotaNftKvPairVersion{
					BlockNumber: define.BlockNumber,
					CotaId:      define.CotaId,
					Total:       define.Total,
					Issued:      define.Issued,
					OldIssued:   define.Issued,
					Configure:   define.Configure,
					LockHash:    define.LockHash,
					TxIndex:     define.TxIndex,
					ActionType:  0,
				}
				defineCotaVersions[i] = defineCotaVersion
			}
			// create define cotas versions
			if err := tx.Model(DefineCotaNftKvPairVersion{}).WithContext(ctx).Create(defineCotaVersions).Error; err != nil {
				return err
			}
		}
		if kvPair.HasUpdatedDefineCotas() {
			updatedDefineCotaVersions := make([]DefineCotaNftKvPairVersion, len(kvPair.UpdatedDefineCotas))
			for i, define := range kvPair.UpdatedDefineCotas {
				var defineCota DefineCotaNftKvPair
				if err := tx.Model(DefineCotaNftKvPair{}).WithContext(ctx).Where("cota_id = ?", define.CotaId).First(&defineCota).Error; err != nil {
					return err
				}
				defineCotaVersion := DefineCotaNftKvPairVersion{
					OldBlockNumber: defineCota.BlockNumber,
					BlockNumber:    define.BlockNumber,
					CotaId:         define.CotaId,
					Total:          define.Total,
					Issued:         define.Issued,
					OldIssued:      defineCota.Issued,
					Configure:      define.Configure,
					LockHash:       define.LockHash,
					TxIndex:        define.TxIndex,
					ActionType:     1,
				}
				updatedDefineCotaVersions[i] = defineCotaVersion
			}
			// create updated define cotas versions
			if err := tx.Model(DefineCotaNftKvPairVersion{}).WithContext(ctx).Create(updatedDefineCotaVersions).Error; err != nil {
				return err
			}
			// update define cotas
			updatedDefineCotas := make([]DefineCotaNftKvPair, len(kvPair.UpdatedDefineCotas))
			for i, cota := range kvPair.UpdatedDefineCotas {
				updatedDefineCotas[i] = DefineCotaNftKvPair{
					BlockNumber: cota.BlockNumber,
					CotaId:      cota.CotaId,
					Total:       cota.Total,
					Issued:      cota.Issued,
					Configure:   cota.Configure,
					LockHash:    cota.LockHash,
					LockHashCRC: cota.LockHashCRC,
					UpdatedAt:   cota.UpdatedAt,
				}
			}
			if err := tx.Model(DefineCotaNftKvPair{}).WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "cota_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"issued", "block_number", "updated_at"}),
			}).Create(updatedDefineCotas).Error; err != nil {
				return err
			}
		}
		if kvPair.HasWithdrawCotas() {
			// create withdraw cotas
			withdrawCotas := make([]WithdrawCotaNftKvPair, len(kvPair.WithdrawCotas))
			for i, cota := range kvPair.WithdrawCotas {
				withdrawCotas[i] = WithdrawCotaNftKvPair{
					BlockNumber:          cota.BlockNumber,
					CotaId:               cota.CotaId,
					CotaIdCRC:            cota.CotaIdCRC,
					TokenIndex:           cota.TokenIndex,
					OutPoint:             cota.OutPoint,
					OutPointCrc:          cota.OutPointCrc,
					TxHash:               cota.TxHash,
					State:                cota.State,
					Configure:            cota.Configure,
					Characteristic:       cota.Characteristic,
					ReceiverLockScriptId: cota.ReceiverLockScriptId,
					LockHash:             cota.LockHash,
					LockHashCrc:          cota.LockHashCrc,
					LockScriptId:         cota.LockScriptId,
					Version:              cota.Version,
				}
			}
			if err := tx.Model(WithdrawCotaNftKvPair{}).WithContext(ctx).Create(withdrawCotas).Error; err != nil {
				return err
			}
			holdCotasSize := len(kvPair.WithdrawCotas)
			removedHoldCotas := make([]biz.HoldCotaNftKvPair, holdCotasSize)
			removedHoldCotaIds := make([]uint, holdCotasSize)
			for i, withdrawCota := range kvPair.WithdrawCotas {
				var holdCota biz.HoldCotaNftKvPair
				if err := tx.Model(HoldCotaNftKvPair{}).WithContext(ctx).Select("*").Where("cota_id = ? and token_index = ?", withdrawCota.CotaId, withdrawCota.TokenIndex).Find(&holdCota).Error; err != nil {
					return err
				}
				// 上面把对象初始化出来了，所以需要通过具体值来判断是否存在
				if holdCota.CotaId == "" {
					continue
				}
				removedHoldCotas[i] = holdCota
				removedHoldCotaIds[i] = holdCota.ID
			}
			if removedHoldCotas[0].CotaId != "" {
				removedHoldCotaVersions := make([]HoldCotaNftKvPairVersion, holdCotasSize)
				blockNumber := kvPair.WithdrawCotas[0].BlockNumber
				for i, cota := range removedHoldCotas {
					removedHoldCotaVersions[i] = HoldCotaNftKvPairVersion{
						OldBlockNumber:    cota.BlockNumber,
						BlockNumber:       blockNumber,
						CotaId:            cota.CotaId,
						TokenIndex:        cota.TokenIndex,
						OldState:          cota.State,
						Configure:         cota.Configure,
						OldCharacteristic: cota.Characteristic,
						OldLockHash:       cota.LockHash,
						TxIndex:           cota.TxIndex,
						ActionType:        2,
					}
				}
				// create removed hold cota versions
				if err := tx.Model(HoldCotaNftKvPairVersion{}).WithContext(ctx).Create(removedHoldCotaVersions).Error; err != nil {
					return err
				}
				// remove those hold cotas that are equal with withdraw cotas
				if err := tx.Model(HoldCotaNftKvPair{}).WithContext(ctx).Delete(&removedHoldCotas, removedHoldCotaIds).Error; err != nil {
					return err
				}
			}
		}
		if kvPair.HasHoldCotas() {
			// create hold cotas
			holdCotas := make([]HoldCotaNftKvPair, len(kvPair.HoldCotas))
			for i, cota := range kvPair.HoldCotas {
				holdCotas[i] = HoldCotaNftKvPair{
					BlockNumber:    cota.BlockNumber,
					CotaId:         cota.CotaId,
					TokenIndex:     cota.TokenIndex,
					State:          cota.State,
					Configure:      cota.Configure,
					Characteristic: cota.Characteristic,
					LockHash:       cota.LockHash,
					LockHashCRC:    cota.LockHashCRC,
				}
			}
			if err := tx.Model(HoldCotaNftKvPair{}).WithContext(ctx).Create(holdCotas).Error; err != nil {
				return err
			}
			newHoldCotaVersions := make([]HoldCotaNftKvPairVersion, len(kvPair.HoldCotas))
			for i, cota := range kvPair.HoldCotas {
				newHoldCotaVersions[i] = HoldCotaNftKvPairVersion{
					BlockNumber:    cota.BlockNumber,
					CotaId:         cota.CotaId,
					TokenIndex:     cota.TokenIndex,
					State:          cota.State,
					Configure:      cota.Configure,
					Characteristic: cota.Characteristic,
					LockHash:       cota.LockHash,
					TxIndex:        cota.TxIndex,
					ActionType:     0,
				}
			}
			// create hold cota versions
			if err := tx.Model(HoldCotaNftKvPairVersion{}).WithContext(ctx).Create(newHoldCotaVersions).Error; err != nil {
				return err
			}
		}
		if kvPair.HasUpdatedHoldCotas() {
			updatedHoldCotaVersions := make([]HoldCotaNftKvPairVersion, len(kvPair.UpdatedHoldCotas))
			for i, cota := range kvPair.UpdatedHoldCotas {
				var oldHoldCota HoldCotaNftKvPair
				if err := tx.Model(HoldCotaNftKvPair{}).WithContext(ctx).Where("cota_id = ? and token_index = ?", cota.CotaId, cota.TokenIndex).First(&oldHoldCota).Error; err != nil {
					return err
				}
				updatedHoldCotaVersions[i] = HoldCotaNftKvPairVersion{
					OldBlockNumber:    oldHoldCota.BlockNumber,
					BlockNumber:       cota.BlockNumber,
					CotaId:            cota.CotaId,
					TokenIndex:        cota.TokenIndex,
					OldState:          oldHoldCota.State,
					State:             cota.State,
					Configure:         cota.Configure,
					OldCharacteristic: oldHoldCota.Characteristic,
					Characteristic:    cota.Characteristic,
					OldLockHash:       oldHoldCota.LockHash,
					LockHash:          cota.LockHash,
					TxIndex:           cota.TxIndex,
					ActionType:        1,
				}
			}
			// create updated hold cotas versions
			if err := tx.Model(HoldCotaNftKvPairVersion{}).WithContext(ctx).Create(updatedHoldCotaVersions).Error; err != nil {
				return err
			}
			// update hold cotas
			updatedHoldCotas := make([]HoldCotaNftKvPair, len(kvPair.UpdatedHoldCotas))
			for i, cota := range kvPair.UpdatedHoldCotas {
				updatedHoldCotas[i] = HoldCotaNftKvPair{
					BlockNumber:    cota.BlockNumber,
					CotaId:         cota.CotaId,
					TokenIndex:     cota.TokenIndex,
					State:          cota.State,
					Configure:      cota.Configure,
					Characteristic: cota.Characteristic,
					LockHash:       cota.LockHash,
					LockHashCRC:    cota.LockHashCRC,
					UpdatedAt:      cota.UpdatedAt,
				}
			}
			if err := tx.Debug().Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "cota_id"}, {Name: "token_index"}},
				DoUpdates: clause.AssignmentColumns([]string{"block_number", "state", "characteristic", "lock_hash", "lock_hash_crc", "updated_at"}),
			}).Create(updatedHoldCotas).Error; err != nil {
				return err
			}
		}
		if kvPair.HasClaimedCotas() {
			// create claimed cotas
			claimedCotas := make([]ClaimedCotaNftKvPair, len(kvPair.ClaimedCotas))
			for i, cota := range kvPair.ClaimedCotas {
				claimedCotas[i] = ClaimedCotaNftKvPair{
					BlockNumber: cota.BlockNumber,
					CotaId:      cota.CotaId,
					CotaIdCRC:   cota.CotaIdCRC,
					TokenIndex:  cota.TokenIndex,
					OutPoint:    cota.OutPoint,
					OutPointCrc: cota.OutPointCrc,
					LockHash:    cota.LockHash,
					LockHashCrc: cota.LockHashCrc,
				}
			}
			if err := tx.Model(ClaimedCotaNftKvPair{}).WithContext(ctx).Create(claimedCotas).Error; err != nil {
				return err
			}
		}
		// create check info
		if err := tx.Debug().Model(CheckInfo{}).WithContext(ctx).Create(&CheckInfo{
			BlockNumber: checkInfo.BlockNumber,
			BlockHash:   checkInfo.BlockHash,
			CheckType:   checkInfo.CheckType,
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (rp kvPairRepo) RestoreCotaEntryKvPairs(ctx context.Context, blockNumber uint64) error {
	return rp.data.db.Transaction(func(tx *gorm.DB) error {
		// delete all register cotas by the block number
		if err := tx.WithContext(ctx).Where("block_number = ?", blockNumber).Delete(RegisterCotaKvPair{}).Error; err != nil {
			return err
		}
		// delete all new define cotas by the block number
		if err := tx.WithContext(ctx).Where("block_number = ?", blockNumber).Delete(DefineCotaNftKvPair{}).Error; err != nil {
			return err
		}
		// delete all create define cota versions by the block number
		if err := tx.WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 0).Delete(DefineCotaNftKvPairVersion{}).Error; err != nil {
			return err
		}
		// 把需要回滚的 block 更新过的 define 恢复到更新前的状态，这里按 cota_id 分组取出回滚 block 下第一条
		var updatedDefineCotaVersions []DefineCotaNftKvPairVersion
		if err := tx.WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 1).Group("cota_id").Order("tx_index").Find(&updatedDefineCotaVersions).Error; err != nil {
			return err
		}
		var updatedDefineCotas []DefineCotaNftKvPair
		for _, version := range updatedDefineCotaVersions {
			updatedDefineCotas = append(updatedDefineCotas, DefineCotaNftKvPair{
				BlockNumber: version.OldBlockNumber,
				CotaId:      version.CotaId,
				Total:       version.Total,
				Issued:      version.OldIssued,
				Configure:   version.Configure,
				LockHash:    version.LockHash,
				LockHashCRC: crc32.ChecksumIEEE([]byte(version.LockHash)),
				UpdatedAt:   time.Now().UTC(),
			})
		}
		if len(updatedDefineCotas) > 0 {
			if err := tx.Debug().WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "cota_id"}},
				UpdateAll: true,
			}).Create(updatedDefineCotas).Error; err != nil {
				return err
			}
		}
		// delete all updated define versions by the block number
		if err := tx.WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 1).Delete(DefineCotaNftKvPairVersion{}).Error; err != nil {
			return err
		}
		// delete all withdraw cotas by the block number
		if err := tx.WithContext(ctx).Where("block_number = ?", blockNumber).Delete(WithdrawCotaNftKvPair{}).Error; err != nil {
			return err
		}
		// delete all hold cotas by the block number
		if err := tx.WithContext(ctx).Where("block_number = ?", blockNumber).Delete(HoldCotaNftKvPair{}).Error; err != nil {
			return err
		}
		// delete all created hold cota versions by the block number
		if err := tx.WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 0).Delete(HoldCotaNftKvPairVersion{}).Error; err != nil {
			return err
		}
		// restore all deleted hold cotas by the block number
		var deletedHoldCotaVersions []HoldCotaNftKvPairVersion
		if err := tx.WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 2).Group("cota_id, token_index").Order("tx_index").Find(&deletedHoldCotaVersions).Error; err != nil {
			return err
		}
		var deletedHoldCotas []HoldCotaNftKvPair
		for _, version := range deletedHoldCotaVersions {
			deletedHoldCotas = append(deletedHoldCotas, HoldCotaNftKvPair{
				BlockNumber:    version.OldBlockNumber,
				CotaId:         version.CotaId,
				TokenIndex:     version.TokenIndex,
				State:          version.OldState,
				Configure:      version.Configure,
				Characteristic: version.OldCharacteristic,
				LockHash:       version.OldLockHash,
				LockHashCRC:    crc32.ChecksumIEEE([]byte(version.OldLockHash)),
			})
		}
		if len(deletedHoldCotas) > 0 {
			if err := tx.WithContext(ctx).Create(deletedHoldCotas).Error; err != nil {
				return err
			}
		}
		// delete all deleted hold cota versions by the block number
		if err := tx.WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 2).Delete(HoldCotaNftKvPairVersion{}).Error; err != nil {
			return err
		}
		// restore all updated hold cotas by the block number
		var updatedHoldCotaVersions []HoldCotaNftKvPairVersion
		if err := tx.WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 1).Group("cota_id, token_index").Order("tx_index").Find(&updatedHoldCotaVersions).Error; err != nil {
			return err
		}
		var updatedHoldCotas []HoldCotaNftKvPair
		for _, version := range updatedHoldCotaVersions {
			updatedHoldCotas = append(updatedHoldCotas, HoldCotaNftKvPair{
				BlockNumber:    version.OldBlockNumber,
				CotaId:         version.CotaId,
				TokenIndex:     version.TokenIndex,
				State:          version.OldState,
				Configure:      version.Configure,
				Characteristic: version.OldCharacteristic,
				LockHash:       version.OldLockHash,
				LockHashCRC:    crc32.ChecksumIEEE([]byte(version.OldLockHash)),
			})
		}
		if len(updatedHoldCotaVersions) > 0 {
			if err := tx.WithContext(ctx).Create(updatedHoldCotas).Error; err != nil {
				return err
			}
		}
		// delete all updated hold cota versions by the block number
		if err := tx.WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 1).Delete(HoldCotaNftKvPairVersion{}).Error; err != nil {
			return err
		}
		// delete all claimed cotas by the block number
		if err := tx.WithContext(ctx).Where("block_number = ?", blockNumber).Delete(ClaimedCotaNftKvPair{}).Error; err != nil {
			return err
		}
		// delete check info
		if err := tx.Debug().WithContext(ctx).Where("block_number = ? and check_type = ?", blockNumber, biz.SyncBlock).Delete(CheckInfo{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (rp kvPairRepo) CreateMetadataKvPairs(ctx context.Context, checkInfo biz.CheckInfo, kvPair *biz.KvPair) error {
	return rp.data.db.Transaction(func(tx *gorm.DB) error {
		if kvPair.HasIssuerInfos() {
			// save issuer info versions
			issuerInfoVersions := make([]IssuerInfoVersion, len(kvPair.IssuerInfos))
			for i, info := range kvPair.IssuerInfos {
				var oldInfo IssuerInfo
				err := tx.Model(IssuerInfo{}).WithContext(ctx).Where("lock_hash = ?", info.LockHash).First(&oldInfo).Error
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
					issuerInfoVersions[i] = IssuerInfoVersion{
						BlockNumber:  info.BlockNumber,
						LockHash:     info.LockHash,
						Version:      info.Version,
						Name:         info.Name,
						Avatar:       info.Avatar,
						Description:  info.Description,
						Localization: info.Localization,
						ActionType:   0,
						TxIndex:      info.TxIndex,
					}
				} else {
					issuerInfoVersions[i] = IssuerInfoVersion{
						OldBlockNumber:  oldInfo.BlockNumber,
						BlockNumber:     info.BlockNumber,
						LockHash:        info.LockHash,
						OldVersion:      oldInfo.Version,
						Version:         info.Version,
						OldName:         oldInfo.Name,
						Name:            info.Name,
						OldAvatar:       oldInfo.Avatar,
						Avatar:          info.Avatar,
						OldDescription:  oldInfo.Description,
						Description:     info.Description,
						OldLocalization: oldInfo.Localization,
						Localization:    info.Localization,
						ActionType:      1,
						TxIndex:         info.TxIndex,
					}
				}
			}
			if err := tx.Model(IssuerInfoVersion{}).WithContext(ctx).Create(issuerInfoVersions).Error; err != nil {
				return err
			}
			// insert issuer info
			issuerInfos := make([]IssuerInfo, len(kvPair.IssuerInfos))
			for i, issuer := range kvPair.IssuerInfos {
				issuerInfos[i] = IssuerInfo{
					BlockNumber:  issuer.BlockNumber,
					LockHash:     issuer.LockHash,
					Version:      issuer.Version,
					Name:         issuer.Name,
					Avatar:       issuer.Avatar,
					Description:  issuer.Description,
					Localization: issuer.Localization,
				}
			}
			if err := tx.Model(IssuerInfo{}).WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "lock_hash"}},
				UpdateAll: true,
			}).Create(issuerInfos).Error; err != nil {
				return err
			}
		}
		if kvPair.HasClassInfos() {
			// save class info versions
			classInfoVersions := make([]ClassInfoVersion, len(kvPair.ClassInfos))
			for i, info := range kvPair.ClassInfos {
				var oldInfo ClassInfo
				err := tx.Model(ClassInfo{}).WithContext(ctx).Where("cota_id = ?", info.CotaId).First(&oldInfo).Error
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
					classInfoVersions[i] = ClassInfoVersion{
						BlockNumber:    0,
						CotaId:         info.CotaId,
						Version:        info.Version,
						Name:           info.Name,
						Symbol:         info.Symbol,
						Description:    info.Description,
						Image:          info.Image,
						Audio:          info.Audio,
						Video:          info.Video,
						Model:          info.Model,
						Characteristic: info.Characteristic,
						Properties:     info.Properties,
						Localization:   info.Localization,
						ActionType:     0,
						TxIndex:        info.TxIndex,
					}
				} else {
					classInfoVersions[i] = ClassInfoVersion{
						OldBlockNumber:    oldInfo.BlockNumber,
						BlockNumber:       info.BlockNumber,
						CotaId:            info.CotaId,
						OldVersion:        oldInfo.Version,
						Version:           info.Version,
						OldName:           oldInfo.Name,
						Name:              info.Name,
						OldSymbol:         oldInfo.Symbol,
						Symbol:            info.Symbol,
						OldDescription:    oldInfo.Description,
						Description:       info.Description,
						OldImage:          oldInfo.Image,
						Image:             info.Image,
						OldAudio:          oldInfo.Audio,
						Audio:             info.Audio,
						OldVideo:          oldInfo.Video,
						Video:             info.Video,
						OldModel:          oldInfo.Model,
						Model:             info.Model,
						OldCharacteristic: oldInfo.Characteristic,
						Characteristic:    info.Characteristic,
						OldProperties:     oldInfo.Properties,
						Properties:        info.Properties,
						OldLocalization:   oldInfo.Localization,
						Localization:      info.Localization,
						ActionType:        1,
						TxIndex:           info.TxIndex,
					}
				}
			}

			if err := tx.Model(ClassInfoVersion{}).WithContext(ctx).Create(classInfoVersions).Error; err != nil {
				return err
			}
			// insert class info
			classInfos := make([]ClassInfo, len(kvPair.ClassInfos))
			for i, class := range kvPair.ClassInfos {
				classInfos[i] = ClassInfo{
					BlockNumber:    class.BlockNumber,
					CotaId:         class.CotaId,
					Version:        class.Version,
					Name:           class.Name,
					Symbol:         class.Symbol,
					Description:    class.Description,
					Image:          class.Image,
					Audio:          class.Audio,
					Video:          class.Video,
					Model:          class.Model,
					Characteristic: class.Characteristic,
					Properties:     class.Properties,
					Localization:   class.Localization,
				}
			}
			if err := tx.Model(ClassInfo{}).WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "cota_id"}},
				UpdateAll: true,
			}).Create(classInfos).Error; err != nil {
				return err
			}
		}
		// create check info
		if err := tx.Debug().Model(CheckInfo{}).WithContext(ctx).Create(&CheckInfo{
			BlockNumber: checkInfo.BlockNumber,
			BlockHash:   checkInfo.BlockHash,
			CheckType:   checkInfo.CheckType,
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (rp kvPairRepo) RestoreMetadataKvPairs(ctx context.Context, blockNumber uint64) error {
	return rp.data.db.Transaction(func(tx *gorm.DB) error {
		// 删掉所有新建的 issuer info
		if err := tx.WithContext(ctx).Where("block_number = ?", blockNumber).Delete(IssuerInfo{}).Error; err != nil {
			return err
		}
		// 把需要回滚的 block 更新过的 issuerInfo 恢复到更新前的状态
		var issuerInfoVersions []IssuerInfoVersion
		if err := tx.Model(IssuerInfoVersion{}).WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 1).Group("lock_hash").Order("tx_index").Find(&issuerInfoVersions).Error; err != nil {
			return err
		}
		var updatedIssuerInfos []IssuerInfo
		for _, version := range issuerInfoVersions {
			updatedIssuerInfos = append(updatedIssuerInfos, IssuerInfo{
				BlockNumber:  version.OldBlockNumber,
				LockHash:     version.LockHash,
				Version:      version.OldVersion,
				Name:         version.OldName,
				Avatar:       version.OldAvatar,
				Description:  version.OldDescription,
				Localization: version.OldLocalization,
				UpdatedAt:    time.Now().UTC(),
			})
		}
		if len(updatedIssuerInfos) > 0 {
			if err := tx.Model(IssuerInfo{}).WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "lock_hash"}},
				UpdateAll: true,
			}).Create(updatedIssuerInfos).Error; err != nil {
				return err
			}
		}
		// delete all class info by the block number
		if err := tx.Debug().WithContext(ctx).Where("block_number = ?", blockNumber).Delete(ClassInfo{}).Error; err != nil {
			return err
		}
		var classInfoVersions []ClassInfoVersion
		if err := tx.Model(ClassInfoVersion{}).WithContext(ctx).Where("block_number = ? and action_type = ?", blockNumber, 1).Group("cota_id").Order("tx_index").Find(&classInfoVersions).Error; err != nil {
			return err
		}
		var updatedClassInfos []ClassInfo
		for _, version := range classInfoVersions {
			updatedClassInfos = append(updatedClassInfos, ClassInfo{
				BlockNumber:    version.OldBlockNumber,
				CotaId:         version.CotaId,
				Version:        version.OldVersion,
				Name:           version.OldName,
				Symbol:         version.OldSymbol,
				Description:    version.OldDescription,
				Image:          version.OldImage,
				Audio:          version.OldAudio,
				Video:          version.OldVideo,
				Model:          version.OldModel,
				Characteristic: version.OldCharacteristic,
				Properties:     version.OldProperties,
				Localization:   version.OldLocalization,
				UpdatedAt:      time.Now().UTC(),
			})
		}
		if len(updatedClassInfos) > 0 {
			if err := tx.Debug().Model(ClassInfo{}).WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "cota_id"}},
				UpdateAll: true,
			}).Create(updatedClassInfos).Error; err != nil {
				return err
			}
		}
		// delete check info
		if err := tx.Debug().WithContext(ctx).Where("block_number = ? and check_type = ?", blockNumber, biz.SyncMetadata).Delete(CheckInfo{}).Error; err != nil {
			return err
		}
		return nil
	})
}
