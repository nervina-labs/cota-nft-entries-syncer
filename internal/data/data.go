package data

import (
	"context"
	"encoding/hex"
	"errors"
	"os"

	"github.com/golang-migrate/migrate/v4"
	mMsql "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/wire"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/config"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	ckbTypes "github.com/nervosnetwork/ckb-sdk-go/types"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers
var ProviderSet = wire.NewSet(NewData, NewDBMigration, NewCheckInfoRepo, NewRegisterCotaKvPairRepo,
	NewDefineCotaNftKvPairRepo, NewHoldCotaNftKvPairRepo, NewWithdrawCotaNftKvPairRepo, NewClaimedCotaNftKvPairRepo,
	NewKvPairRepo, NewSystemScripts, NewCkbNodeClient, NewBlockSyncer, NewMetadataSyncer, NewCotaWitnessArgsParser,
	NewMintCotaKvPairRepo, NewTransferCotaKvPairRepo, NewIssuerInfoRepo, NewClassInfoRepo, NewInvalidDateRepo, NewWithdrawExtraInfoRepo)

type Data struct {
	db *gorm.DB
}

type Option func(*Data)

func NewData(conf *config.Database, logger *logger.Logger) (*Data, func(), error) {
	dsn := os.Getenv("DATABASE_URL")
	logger.Error(context.TODO(), "dsn", dsn)
	if dsn == "" {
		dsn = conf.Dsn
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Errorf(context.TODO(), "failed opening connection to mysql: %v", err)
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf(context.TODO(), "failed get sql db: %v", err)
		return nil, nil, err
	}
	sqlDB.SetMaxIdleConns(conf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(conf.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(conf.ConnMaxLifeTime)

	return &Data{
			db: db,
		}, func() {
			if err := sqlDB.Close(); err != nil {
				logger.Error(context.TODO(), err)
			}
			logger.Info(context.TODO(), "successfully closed the database")
		}, nil
}

type CkbNodeClient struct {
	Rpc  rpc.Client
	Mode string
}

func NewCkbNodeClient(conf *config.CkbNode, logger *logger.Logger) (*CkbNodeClient, error) {
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		rpcURL = conf.RpcUrl
	}

	client, err := rpc.Dial(rpcURL)
	if err != nil {
		logger.Errorf(context.TODO(), "failed to connect to the ckb node")
		return nil, err
	}
	return &CkbNodeClient{
		Rpc:  client,
		Mode: conf.Mode,
	}, nil
}

type DBMigration struct {
	data   *Data
	logger *logger.Logger
}

func (m *DBMigration) Up() error {
	sqlDB, err := m.data.db.DB()
	if err != nil {
		m.logger.Errorf(context.TODO(), "failed get sql db: %v", err)
		return err
	}
	driver, err := mMsql.WithInstance(sqlDB, &mMsql.Config{})
	if err != nil {
		return err
	}
	migration, err := migrate.NewWithDatabaseInstance("file://./internal/db/migrations", m.data.db.Migrator().CurrentDatabase(), driver)
	if err != nil {
		return err
	}
	err = migration.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		m.logger.Infof(context.TODO(), "migration failed: %v", err)
		return err
	}
	return nil
}

type SystemScriptOption func(o *SystemScripts)

type SystemScript struct {
	CodeHash ckbTypes.Hash
	HashType ckbTypes.ScriptHashType
	Args     []byte
	OutPoint ckbTypes.OutPoint
	DepType  ckbTypes.DepType
}

type SystemScripts struct {
	CotaRegistryType SystemScript
	CotaType         SystemScript
}

func NewSystemScripts(client *CkbNodeClient, logger *logger.Logger) SystemScripts {
	chainInfo, err := client.Rpc.GetBlockchainInfo(context.Background())
	if err != nil {
		logger.Fatalf(context.Background(), "RPC get_blockchain_info error")
	}
	return SystemScripts{
		CotaRegistryType: cotaRegistryScript(chainInfo.Chain),
		CotaType:         cotaTypeScript(chainInfo.Chain),
	}
}

func cotaRegistryScript(chain string) SystemScript {
	if chain == "ckb" {
		args, _ := hex.DecodeString("563631b49cee549f3585ab4dde5f9d590f507f1f")
		return SystemScript{
			CodeHash: ckbTypes.HexToHash("0x90ca618be6c15f5857d3cbd09f9f24ca6770af047ba9ee70989ec3b229419ac7"),
			HashType: ckbTypes.HashTypeType,
			Args:     args,
			OutPoint: ckbTypes.OutPoint{
				TxHash: ckbTypes.HexToHash("0x42a5b04df6ff0e2819ec6b814d33a028ed1593bc9e1cca463f679af555dce106"),
				Index:  0,
			},
			DepType: ckbTypes.DepTypeDepGroup,
		}
	}
	args, _ := hex.DecodeString("f9910364e0ca81a0e074f3aa42fe78cfcc880da6")
	return SystemScript{
		CodeHash: ckbTypes.HexToHash("0x9302db6cc1344b81a5efee06962abcb40427ecfcbe69d471b01b2658ed948075"),
		HashType: ckbTypes.HashTypeType,
		Args:     args,
		OutPoint: ckbTypes.OutPoint{
			TxHash: ckbTypes.HexToHash("0x698f2a29021ebd741b4a38b4a5f8fa3686103ba66773e7549b204a67db015ba0"),
			Index:  0,
		},
		DepType: ckbTypes.DepTypeDepGroup,
	}
}

func cotaTypeScript(chain string) SystemScript {
	if chain == "ckb" {
		return SystemScript{
			CodeHash: ckbTypes.HexToHash("0x1122a4fb54697cf2e6e3a96c9d80fd398a936559b90954c6e88eb7ba0cf652df"),
			HashType: ckbTypes.HashTypeType,
			OutPoint: ckbTypes.OutPoint{
				TxHash: ckbTypes.HexToHash("0x42a5b04df6ff0e2819ec6b814d33a028ed1593bc9e1cca463f679af555dce106"),
				Index:  0,
			},
			DepType: ckbTypes.DepTypeDepGroup,
		}
	}
	return SystemScript{
		CodeHash: ckbTypes.HexToHash("0x89cd8003a0eaf8e65e0c31525b7d1d5c1becefd2ea75bb4cff87810ae37764d8"),
		HashType: ckbTypes.HashTypeType,
		OutPoint: ckbTypes.OutPoint{
			TxHash: ckbTypes.HexToHash("0x698f2a29021ebd741b4a38b4a5f8fa3686103ba66773e7549b204a67db015ba0"),
			Index:  0,
		},
		DepType: ckbTypes.DepTypeDepGroup,
	}
}

func (m *DBMigration) Down() error {
	sqlDB, err := m.data.db.DB()
	if err != nil {
		m.logger.Errorf(context.TODO(), "failed get sql db: %v", err)
		return err
	}
	driver, err := mMsql.WithInstance(sqlDB, &mMsql.Config{})
	if err != nil {
		return err
	}
	migration, err := migrate.NewWithDatabaseInstance("file://../db/migrations", m.data.db.Migrator().CurrentDatabase(), driver)
	if err != nil {
		return err
	}
	return migration.Down()
}

func NewDBMigration(data *Data, logger *logger.Logger) *DBMigration {
	return &DBMigration{
		data:   data,
		logger: logger,
	}
}
