package queue

import (
	"github.com/andrewpillar/thrall/errors"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/backends/result"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
)

var (
	servers = make(map[string]*machinery.Server)

	Builds = "thrall_builds"
)

func New(name, addr, password string) (*machinery.Server, error) {
	url := "redis://"

	if password != "" {
		url += password + "@"
	}

	url += addr

	cfg := &config.Config{
		Broker:        url,
		DefaultQueue:  name,
		ResultBackend: url,
	}

	server, err := machinery.NewServer(cfg)

	if err != nil {
		return nil, errors.Err(err)
	}

	servers[name] = server

	return server, nil
}

func SendTask(name string, sig *tasks.Signature) (*result.AsyncResult, error) {
	server, ok := servers[name]

	if !ok {
		return nil, errors.Err(errors.New("unknown queue server " + name))
	}

	r, err := server.SendTask(sig)

	return r, errors.Err(err)
}
