package mysql

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/venus-market/models/repo"
	mtypes "github.com/filecoin-project/venus-messager/types"
	"golang.org/x/xerrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type storageAsk struct {
	ID            uint       `gorm:"primary_key"`
	Miner         Address    `gorm:"column:miner;type:varchar(128);uniqueIndex"`
	Price         mtypes.Int `gorm:"column:price;type:varchar(256);"`
	VerifiedPrice mtypes.Int `gorm:"column:verified_price;type:varchar(256);"`
	MinPieceSize  int64      `gorm:"column:min_piece_size;type:bigint;"`
	MaxPieceSize  int64      `gorm:"column:max_piece_size;type:bigint;"`
	Timestamp     int64      `gorm:"column:timestamp;type:bigint;"`
	Expiry        int64      `gorm:"column:expiry;type:bigint;"`
	SeqNo         uint64     `gorm:"column:seq_no;type:bigint unsigned;"`
	Signature     Signature  `gorm:"column:signature;type:blob;"`
	TimeStampOrm
}

func (a *storageAsk) TableName() string {
	return "storage_asks"
}

func fromStorageAsk(src *storagemarket.SignedStorageAsk) *storageAsk {
	ask := &storageAsk{}
	if src.Ask != nil {
		ask.Miner = toAddress(src.Ask.Miner)
		ask.Price = convertBigInt(src.Ask.Price)
		ask.VerifiedPrice = convertBigInt(src.Ask.VerifiedPrice)
		ask.MinPieceSize = int64(src.Ask.MinPieceSize)
		ask.MaxPieceSize = int64(src.Ask.MaxPieceSize)
		ask.Timestamp = int64(src.Ask.Timestamp)
		ask.Expiry = int64(src.Ask.Expiry)
		ask.SeqNo = src.Ask.SeqNo
	}
	if src.Signature != nil {
		ask.Signature = Signature{
			Type: src.Signature.Type,
			Data: src.Signature.Data,
		}
	}

	return ask
}

func toStorageAsk(src *storageAsk) (*storagemarket.SignedStorageAsk, error) {
	ask := &storagemarket.SignedStorageAsk{
		Ask: &storagemarket.StorageAsk{
			Miner:         src.Miner.addr(),
			Price:         abi.TokenAmount{Int: src.Price.Int},
			VerifiedPrice: abi.TokenAmount{Int: src.VerifiedPrice.Int},
			MinPieceSize:  abi.PaddedPieceSize(src.MinPieceSize),
			MaxPieceSize:  abi.PaddedPieceSize(src.MaxPieceSize),
			Timestamp:     abi.ChainEpoch(src.Timestamp),
			Expiry:        abi.ChainEpoch(src.Expiry),
			SeqNo:         src.SeqNo,
		},
	}
	if len(src.Signature.Data) != 0 {
		ask.Signature = &crypto.Signature{
			Type: src.Signature.Type,
			Data: src.Signature.Data,
		}
	}

	return ask, nil
}

type storageAskRepo struct {
	*gorm.DB
}

func NewStorageAskRepo(db *gorm.DB) *storageAskRepo {
	return &storageAskRepo{db}
}

func (a *storageAskRepo) GetAsk(miner address.Address) (*storagemarket.SignedStorageAsk, error) {
	var res storageAsk
	err := a.DB.Take(&res, "miner = ?", cutPrefix(miner)).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repo.ErrNotFound
		}
	}
	return toStorageAsk(&res)
}

func (a *storageAskRepo) SetAsk(ask *storagemarket.SignedStorageAsk) error {
	if ask == nil || ask.Ask == nil {
		return xerrors.Errorf("param is nil")
	}
	return a.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "miner"}},
		UpdateAll: true,
	}).Save(fromStorageAsk(ask)).Error
}
