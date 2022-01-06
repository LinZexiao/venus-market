package clients

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-market/config"
	"github.com/filecoin-project/venus-market/utils"
	v1api "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	"github.com/ipfs-force-community/venus-common-utils/apiinfo"
	"github.com/ipfs-force-community/venus-common-utils/metrics"
	"go.uber.org/fx"
)

func NodeClient(mctx metrics.MetricsCtx, lc fx.Lifecycle, nodeCfg *config.Node) (v1api.FullNode, error) {
	fullNode := v1api.FullNodeStruct{}

	aInfo := apiinfo.NewAPIInfo(nodeCfg.Url, nodeCfg.Token)
	addr, err := aInfo.DialArgs("v1")
	if err != nil {
		return nil, err
	}

	closer, err := jsonrpc.NewMergeClient(mctx, addr, "Filecoin", utils.GetInternalStructs(&fullNode), aInfo.AuthHeader())

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			closer()
			return nil
		},
	})
	return &fullNode, err
}
