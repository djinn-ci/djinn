package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/andrewpillar/cli"

	"github.com/andrewpillar/thrall/config"
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Thrall CI server\n"))
}

func mainCommand(c cli.Command) {
	f, err := os.Open(c.Flags.GetString("config"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	server, err := config.DecodeServer(f)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	servers := make([]*http.Server, 0)

	httpServer := &http.Server{
		Addr:         server.Listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      http.HandlerFunc(handler),
	}

	servers = append(servers, httpServer)

	if server.SSL.Cert != "" && server.SSL.Key != "" {
		httpsServer := &http.Server{
			Addr:         server.SSL.Listen,
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      http.HandlerFunc(handler),
		}

		servers = append(servers, httpsServer)

		go func() {
			if err := httpsServer.ListenAndServeTLS(server.SSL.Cert, server.SSL.Key); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
		}()
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
	}()

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt)

	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second * 15))
	defer cancel()

	for _, s := range servers {
		s.Shutdown(ctx)
	}

	fmt.Printf("shutting down\n")
}

func main() {
	c := cli.New()

	cmd := c.Main(mainCommand)

	cmd.AddFlag(&cli.Flag{
		Name:     "config",
		Short:    "-c",
		Long:     "--config",
		Argument: true,
		Default:  "thrall-server.yml",
	})

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
