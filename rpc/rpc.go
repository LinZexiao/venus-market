package rpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/filecoin-project/venus-auth/jwtclient"

	"github.com/filecoin-project/venus-market/v2/config"
)

var log = logging.Logger("modules")

func ServeRPC(ctx context.Context, home config.IHome, apiCfg *config.API, mux *mux.Router, maxRequestSize int64,
	namespace string, authClient *jwtclient.AuthClient, api interface{}, shutdownCh <-chan struct{},
) error {
	serverOptions := make([]jsonrpc.ServerOption, 0)
	if maxRequestSize != 0 { // config set
		serverOptions = append(serverOptions, jsonrpc.WithMaxRequestSize(maxRequestSize))
	}

	rpcServer := jsonrpc.NewServer(serverOptions...)
	rpcServer.Register(namespace, api)
	mux.Handle("/rpc/v0", rpcServer)
	mux.PathPrefix("/").Handler(http.DefaultServeMux)

	localJwtClient, err := getLocalJwtClient(home, apiCfg)
	if err != nil {
		return err
	}
	var handler http.Handler
	if authClient != nil {
		handler = jwtclient.NewAuthMux(localJwtClient, jwtclient.WarpIJwtAuthClient(authClient), mux)
	} else {
		handler = jwtclient.NewAuthMux(localJwtClient, nil, mux)
	}
	srv := &http.Server{Handler: handler}

	go func() {
		select {
		case <-shutdownCh:
		case <-ctx.Done():
		}
		log.Warn("RPC Shutting down...")
		if err := srv.Shutdown(context.Background()); err != nil && err != http.ErrServerClosed {
			log.Errorf("shutting down RPC server failed: %s", err)
		}
		log.Warn("RPC Graceful shutdown successful")
	}()

	addr, err := multiaddr.NewMultiaddr(apiCfg.ListenAddress)
	if err != nil {
		return err
	}

	nl, err := manet.Listen(addr)
	if err != nil {
		return err
	}
	log.Infof("start rpc listen %s", addr)

	if err := srv.Serve(manet.NetListener(nl)); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func getLocalJwtClient(home config.IHome, apiCfg *config.API) (jwtclient.IJwtAuthClient, error) {
	if len(apiCfg.PrivateKey) == 0 {
		secret, err := jwtclient.RandSecret()
		if err != nil {
			return nil, err
		}
		apiCfg.PrivateKey = hex.EncodeToString(secret)
		err = config.SaveConfig(home)
		if err != nil {
			return nil, err
		}
	}

	secret, err := hex.DecodeString(apiCfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	localJwtClient, token, err := jwtclient.NewLocalAuthClientWithSecret(secret)
	if err != nil {
		return nil, err
	}

	err = saveAPIInfo(home, apiCfg, token)
	if err != nil {
		return nil, err
	}
	return localJwtClient, nil
}

func saveAPIInfo(home config.IHome, apiCfg *config.API, token []byte) error {
	homePath, err := home.HomePath()
	if err != nil {
		return fmt.Errorf("unable to home path to save api/token")
	}
	_ = ioutil.WriteFile(path.Join(string(homePath), "api"), []byte(apiCfg.ListenAddress), 0o644)
	_ = ioutil.WriteFile(path.Join(string(homePath), "token"), token, 0o644)
	return nil
}
