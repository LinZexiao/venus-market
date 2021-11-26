package clients

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-market/config"
	"github.com/filecoin-project/venus-market/minermgr"
	vCrypto "github.com/filecoin-project/venus/pkg/crypto"
	"github.com/filecoin-project/venus/pkg/wallet"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"github.com/ipfs-force-community/venus-common-utils/metrics"
	"go.uber.org/fx"
	"golang.org/x/xerrors"
)

type MsgMeta struct {
	Type string
	// Additional data related to what is signed. Should be verifiable with the
	// signed bytes (e.g. CID(Extra).Bytes() == toSign)
	Extra []byte
}

type ISinger interface {
	WalletHas(ctx context.Context, addr address.Address) (bool, error)
	WalletSign(ctx context.Context, k address.Address, msg []byte, meta wallet.MsgMeta) (*vCrypto.Signature, error)
}

type WalletClient struct {
	Internal struct {
		WalletHas  func(ctx context.Context, addr address.Address) (bool, error)
		WalletSign func(ctx context.Context, k address.Address, msg []byte, meta wallet.MsgMeta) (*vCrypto.Signature, error)
	}
}

func (walletClient *WalletClient) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	return walletClient.Internal.WalletHas(ctx, addr)
}

func (walletClient *WalletClient) WalletSign(ctx context.Context, k address.Address, msg []byte, meta wallet.MsgMeta) (*vCrypto.Signature, error) {
	return walletClient.Internal.WalletSign(ctx, k, msg, meta)
}

type SignerParams struct {
	fx.In
	SignerCfg *config.Signer
	Mgr       minermgr.IMinerMgr `optional:"true"`
}

func NewISignerClient(mctx metrics.MetricsCtx, lc fx.Lifecycle, params SignerParams) (ISinger, error) {
	var signer ISinger
	var closer jsonrpc.ClientCloser
	var err error
	switch params.SignerCfg.SignerType {
	case "local":
		signer, closer, err = newWalletClient(context.Background(), params.SignerCfg.Token, params.SignerCfg.Url)
	case "gateway":
		signer, closer, err = newGatewayWalletClient(context.Background(), params.Mgr, params.SignerCfg)
	default:
		return nil, xerrors.Errorf("unsupport sign type %s", params.SignerCfg.SignerType)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			if closer != nil {
				closer()
			}
			return nil
		},
	})
	return signer, err
}

func newWalletClient(ctx context.Context, token, url string) (*WalletClient, jsonrpc.ClientCloser, error) {
	apiInfo := apiinfo.NewAPIInfo(url, token)
	addr, err := apiInfo.DialArgs("v0")
	if err != nil {
		return nil, nil, err
	}

	walletClient := WalletClient{}
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin", []interface{}{&walletClient.Internal}, apiInfo.AuthHeader())

	return &walletClient, closer, err
}
