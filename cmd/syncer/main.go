package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/app"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/config"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/data"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/logger"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/service"
	"gopkg.in/natefinch/lumberjack.v2"
)

func newApp(logger *logger.Logger, blockSyncSvc *service.BlockSyncService, checkInfoCleanerSvc *service.CheckInfoCleanerService, metadataSyncSvc *service.MetadataSyncService, invalidDataCleanerSvc *service.InvalidDataCleaner, withdrawExtraInfoService *service.WithdrawExtraInfoService, m *data.DBMigration) *app.App {
	return app.NewApp(
		app.Name("cota-nft-entries-syncer"),
		app.Version("0.0.1"),
		app.Logger(logger),
		app.Services(blockSyncSvc, checkInfoCleanerSvc, metadataSyncSvc, invalidDataCleanerSvc, withdrawExtraInfoService), app.Migration(m))
}

func main() {
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatalf("init.setupConfig err: %v", err)
	}
	dataConf, err := setupDataConf(conf)
	if err != nil {
		log.Fatalf("init.setupAppConfig err: %v", err)
	}
	appConf, err := setupAppConf(conf)
	if err != nil {
		log.Fatalf("init.setupDataConfig err: %v", err)
	}
	ckbNodeConf, err := setupCkbNodeConf(conf)
	if err != nil {
		log.Fatalf("init.setupCkbNodeConfig err: %v", err)
	}
	logger := logger.NewLogger(&lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s%s", appConf.LogSavePath, appConf.LogFileName, appConf.LogFileExt),
		MaxSize:    600,
		MaxAge:     10,
		MaxBackups: 3,
		LocalTime:  true,
	}, "", log.LstdFlags)

	app, cleanup, err := initApp(&dataConf.Database, ckbNodeConf, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	fmt.Printf("pid: %v", os.Getpid())
	if err := app.Run(appConf.Mode); err != nil {
		panic(err)
	}
}

func setupAppConf(conf *config.Config) (*config.App, error) {
	var appConf *config.App
	err := conf.ReadSection("app", &appConf)
	return appConf, err
}

func setupDataConf(conf *config.Config) (*config.Data, error) {
	var dataConf *config.Data
	err := conf.ReadSection("data", &dataConf)
	return dataConf, err
}

func setupCkbNodeConf(conf *config.Config) (*config.CkbNode, error) {
	var ckbNodeConf *config.CkbNode
	err := conf.ReadSection("ckb_node", &ckbNodeConf)
	return ckbNodeConf, err
}
