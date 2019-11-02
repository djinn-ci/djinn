package config

import (
	"io"

	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

type Server struct {
	Host string

	Images    string
	Artifacts string
	Objects   string

	Net struct {
		Listen string

		SSL struct {
			Cert   string
			Key    string
		}
	}

	Crypto struct {
		Hash  string
		Block string
		Salt  string
		Auth  string
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

	Log struct {
		Level  string
		File   string
		Access bool
	}

	Drivers []struct {
		Type  string
		Queue string
	}

	Providers []struct {
		Name         string
		Secret       string
		ClientID     string `toml:"client_id"`
		ClientSecret string `toml:"client_secret"`
	}
}

func DecodeServer(r io.Reader) (Server, error) {
	dec := toml.NewDecoder(r)

	server := Server{}

	if err := dec.Decode(&server); err != nil {
		return server, errors.Err(err)
	}

	return server, nil
}
