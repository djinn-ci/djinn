package config

import (
	"strings"
	"testing"

	"github.com/andrewpillar/config"
)

func Test_DecodeWorker(t *testing.T) {
	r := strings.NewReader(`

pidfile "/run/djinn/worker.pid"

log info "/var/log/djinn/worker.log"

driver "qemu-x86_64"

timeout 30m

provider github {}
provider gitlab {}

crypto {
	block "0000000000000000"
	salt  "0000000000000000"
}

smtp {
	addr  "smtp.example.com:587"
	admin "noreply@djinn-ci.com"
}

database {
	addr "localhost:5432"
	name "djinn"

	username "djinn_worker"
	password "secret"
}

redis {
	addr "localhost:6379"
}

store artifacts {
	type  "file"
	path  "/var/lib/djinn/artifacts"
	limit 50MB
}

store images {
	type "file"
	path "/var/lib/djinn/images"
}

store objects {
	type "file"
	path "/var/lib/djinn/objects"
}
`)

	dec := config.NewDecoder(t.Name(), decodeOpts...)

	var cfg workerCfg

	if err := dec.Decode(&cfg, r); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
}
