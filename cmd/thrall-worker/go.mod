module github.com/andrewpillar/thrall/cmd/thrall-worker

replace github.com/andrewpillar/thrall => ../../

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20181106193140-f5749085e9cb

require (
	github.com/andrewpillar/cli v1.1.0
	github.com/andrewpillar/thrall v0.0.0-00010101000000-000000000000
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/lib/pq v1.1.0
	github.com/opencontainers/image-spec v1.0.1 // indirect
)
