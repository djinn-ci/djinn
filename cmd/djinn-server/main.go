package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/errors"
	"djinn-ci.com/serverutil"
	"djinn-ci.com/version"
)

func main() {
	crypto.CheckCSPRNG()

	api, config, ui, showversion := serverutil.ParseFlags(os.Args)

	if showversion {
		fmt.Printf("%s %s %s/%s\n", os.Args[0], version.Build, runtime.GOOS, runtime.GOARCH)
		return
	}

	srv, cfg, close_, err := serverutil.Init(config)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], errors.Cause(err))
		os.Exit(1)
	}

	defer close_()

	serverutil.RegisterRoutes(cfg, api, ui, srv)
	serverutil.Start(srv)

	c := make(chan os.Signal, 1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*15))
	defer cancel()

	signal.Notify(c, os.Interrupt)

	sig := <-c

	srv.Shutdown(ctx)
	srv.Log.Info.Println("signal:", sig, "received, shutting down")
}
