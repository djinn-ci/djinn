package server

import (
	"context"
	"net/http"
	"time"

	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/web"
)

type Server struct {
	http  *http.Server
	https *http.Server

	HttpAddr  string
	HttpsAddr string

	SSLCert string
	SSLKey  string
}

func (s *Server) Init(h http.Handler) {
	s.http = &http.Server{
		Addr:         s.HttpAddr,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      h,
	}

	if s.HttpsAddr != "" && s.SSLCert != "" && s.SSLKey != "" {
		s.https = &http.Server{
			Addr:         s.HttpsAddr,
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      h,
		}

		s.http.Handler = web.NewSecureRedirect(s.https.Addr)
	}
}

func (s Server) Serve() {
	if s.https != nil {
		go func() {
			if err := s.https.ListenAndServeTLS(s.SSLCert, s.SSLKey); err != nil {
				log.Error.Println("error serving request:", err)
			}
		}()
	}

	go func() {
		if err := s.http.ListenAndServe(); err != nil {
			log.Error.Println("error serving request:", err)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) {
	if s.https != nil {
		s.https.Shutdown(ctx)
	}

	s.http.Shutdown(ctx)
}
