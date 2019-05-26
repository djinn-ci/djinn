module github.com/andrewpillar/thrall/cmd/thrall-server

replace github.com/andrewpillar/thrall => ../../

require (
	github.com/RichardKnop/machinery v1.6.4
	github.com/andrewpillar/cli v1.1.0
	github.com/andrewpillar/thrall v0.0.0-00010101000000-000000000000
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/gorilla/mux v1.7.2
	github.com/gorilla/schema v1.1.0 // indirect
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.1.3 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/speps/go-hashids v2.0.0+incompatible // indirect
	github.com/valyala/quicktemplate v1.1.1 // indirect
)
