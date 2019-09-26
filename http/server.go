package http

import (
	"net/http"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/gorilla/csrf"
)

type Server struct {
	*http.Server

	Addr      string
	Cert      string
	Key       string
	CSRFToken []byte
}

func (s *Server) Init(h http.Handler) {
	if s.CSRFToken != nil {
		h = csrf.Protect(
			s.CSRFToken,
			csrf.RequestHeader("X-CSRF-Token"),
			csrf.FieldName("csrf_token"),
		)(h)
	}

	s.Server = &http.Server{
		Addr:         s.Addr,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      h,
	}
}

func (s Server) Serve() error {
	if s.Cert != "" && s.Key != "" {
		err := s.Server.ListenAndServeTLS(s.Cert, s.Key)

		return errors.Err(err)
	}

	return errors.Err(s.Server.ListenAndServe())
}
