package config

import (
	"io"

	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

// Server represents the configuration used for the thrall-server.
type Server struct {
	Host string

	Images    Storage
	Artifacts Storage
	Objects   Storage

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
	}

	Drivers []struct {
		Type  string
		Queue string
	}

	Providers []struct {
		Name         string
		Secret       string
		Endpoint     string
		ClientID     string `toml:"client_id"`
		ClientSecret string `toml:"client_secret"`
	}
}

type Storage struct {
	Kind  string
	Path  string
	Limit int64
}

// DecodeServer takes the given io.Reader, and decodes the its content into a
// Server, which is then returned.
func DecodeServer(r io.Reader) (Server, error) {
	dec := toml.NewDecoder(r)

	server := Server{}

	if err := dec.Decode(&server); err != nil {
		return server, errors.Err(err)
	}

	if server.Images.Kind == "" {
		server.Images.Kind = "file"
	}
	if server.Objects.Kind == "" {
		server.Objects.Kind = "file"
	}
	if server.Artifacts.Kind == "" {
		server.Artifacts.Kind = "file"
	}
	return server, nil
}
