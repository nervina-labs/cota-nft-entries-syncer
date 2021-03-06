package biz

import (
	"github.com/google/wire"
	ckbTypes "github.com/nervosnetwork/ckb-sdk-go/types"
)

var ProviderSet = wire.NewSet(NewCheckInfoUsecase, NewRegisterCotaKvPairUsecase, NewDefineCotaNftKvPairUsecase,
	NewHoldCotaNftKvPairUsecase, NewWithdrawCotaNftKvPairUsecase, NewClaimedCotaNftKvPairUsecase, NewSyncKvPairUsecase,
	NewMintCotaKvPairUsecase, NewTransferCotaKvPairUsecase, NewIssuerInfoUsecase, NewClassInfoUsecase, NewInvalidDataUsecase)

type Entry struct {
	InputType  []byte
	OutputType []byte
	LockScript *ckbTypes.Script
	TxIndex    uint32
	Version    uint8
}
