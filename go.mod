module github.com/andrewpillar/djinn

go 1.14

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20181106193140-f5749085e9cb

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/DATA-DOG/go-sqlmock v1.4.1
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/andrewpillar/query v0.0.0-20201109182332-81f25b5b309b
	github.com/andrewpillar/webutil v0.0.0-20201112225125-acf161db83db
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gorilla/csrf v1.6.2
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/schema v1.2.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.3.0
	github.com/mcmathja/curlyq v0.4.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pelletier/go-toml v1.6.0
	github.com/pkg/sftp v1.10.1
	github.com/rbcervilla/redisstore v1.1.0
	github.com/speps/go-hashids v2.0.0+incompatible
	github.com/valyala/quicktemplate v1.4.1
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20201031054903-ff519b6c9102 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible // indirect
)
