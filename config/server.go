package config

import (
	"io"

	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

type Server struct {
	Drivers []string

	Net struct {
		Listen string

		SSL struct {
			Cert   string
			Key    string
		}
	}

	Crypto struct {
		Hash string
		Key  string
		Salt string
		Auth string
	}

	Database struct {
		Addr     string
		Name     string
		Username string
		Password string
	}

	Redis struct {
		Addr     string
		Password string
	}

	Artifacts FileStore
	Objects   FileStore

	Log struct {
		Level  string
		File   string
		Access bool
	}
}

type FileStore struct {
	Type  string
	Path  string
	Limit int64
}

func DecodeServer(r io.Reader) (Server, error) {
	dec := toml.NewDecoder(r)

	server := Server{}

	if err := dec.Decode(&server); err != nil {
		return server, errors.Err(err)
	}

	return server, nil
}
