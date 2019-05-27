module github.com/andrewpillar/thrall/cmd/thrall-worker

replace github.com/andrewpillar/thrall => ../../

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20181106193140-f5749085e9cb

require (
	github.com/RichardKnop/machinery v1.6.4
	github.com/andrewpillar/cli v1.1.0
	github.com/andrewpillar/thrall v0.0.0-00010101000000-000000000000
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gorilla/schema v1.1.0 // indirect
	github.com/gorilla/sessions v1.1.3 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/kr/fs v0.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/pkg/sftp v1.10.0 // indirect
	github.com/valyala/quicktemplate v1.1.1 // indirect
)
