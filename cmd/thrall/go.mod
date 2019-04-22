module github.com/andrewpillar/thrall/cmd/thrall

replace github.com/andrewpillar/thrall => ../../

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20181106193140-f5749085e9cb

require (
	github.com/Microsoft/go-winio v0.4.11 // indirect
	github.com/andrewpillar/cli v1.1.0
	github.com/andrewpillar/thrall v0.0.0
	github.com/docker/engine v1.13.1 // indirect
	github.com/gogo/protobuf v1.2.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
)
