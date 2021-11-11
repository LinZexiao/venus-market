package sealer

import (
	"context"
	"github.com/filecoin-project/dagstore"
	"github.com/filecoin-project/go-fil-markets/stores"
	"github.com/filecoin-project/venus-market/builder"
	"github.com/filecoin-project/venus-market/config"
	dagstore2 "github.com/filecoin-project/venus-market/dagstore"
	"go.uber.org/fx"
	"golang.org/x/xerrors"
	"os"
	"path/filepath"
	"strconv"
)

var (
	DAGStoreKey = builder.Special{ID: 1}
)

func NewAddressSelector(cfg *config.MarketConfig) (*AddressSelector, error) {
	return &AddressSelector{
		AddressConfig: cfg.AddressConfig,
	}, nil
}

// DAGStore constructs a DAG store using the supplied minerAPI, and the
// user configuration. It returns both the DAGStore and the Wrapper suitable for
// passing to markets.
func NewDAGStore(lc fx.Lifecycle, homeDir *config.HomeDir, cfg *config.DAGStoreConfig, minerAPI dagstore2.MarketAPI) (*dagstore.DAGStore, stores.DAGStoreWrapper, error) {
	// fall back to default root directory if not explicitly set in the config.
	if cfg.RootDir == "" {
		cfg.RootDir = filepath.Join(string(*homeDir), DefaultDAGStoreDir)
	}

	v, ok := os.LookupEnv(EnvDAGStoreCopyConcurrency)
	if ok {
		concurrency, err := strconv.Atoi(v)
		if err == nil {
			cfg.MaxConcurrentReadyFetches = concurrency
		}
	}

	dagst, w, err := dagstore2.NewDAGStore(cfg, minerAPI)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to create DAG store: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return w.Start(ctx)
		},
		OnStop: func(context.Context) error {
			return w.Close()
		},
	})

	return dagst, w, nil
}

var SealerOpts = builder.Options(
	builder.Override(new(*AddressSelector), NewAddressSelector),
	builder.Override(new(dagstore2.MarketAPI), NewMinerAPI),
	builder.Override(DAGStoreKey, NewDAGStore),
)
