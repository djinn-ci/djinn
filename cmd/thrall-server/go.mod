module github.com/andrewpillar/thrall/cmd/thrall-server

replace github.com/andrewpillar/thrall => ../../

require (
	github.com/andrewpillar/cli v1.0.13
	github.com/andrewpillar/thrall v0.0.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.1.3 // indirect
	gopkg.in/boj/redistore.v1 v1.0.0-20160128113310-fc113767cd6b
	gopkg.in/yaml.v2 v2.2.2 // indirect
)
