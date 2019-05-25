module github.com/andrewpillar/thrall/cmd/thrall-server

replace github.com/andrewpillar/thrall => ../../

require (
	github.com/RichardKnop/machinery v1.6.2
	github.com/andrewpillar/cli v1.1.0
	github.com/andrewpillar/thrall v0.0.0
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/go-sql-driver/mysql v1.4.1 // indirect
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/securecookie v1.1.1
	github.com/jmoiron/sqlx v1.2.0
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	golang.org/x/crypto v0.0.0-20190424203555-c05e17bb3b2d // indirect
	google.golang.org/appengine v1.5.0 // indirect
)
