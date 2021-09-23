#!/bin/sh

set -e

_version() {
	git log --decorate=full --format=format:%d |
		head -1 |
		tr ',' '\n' |
		grep tag: |
		cut -d / -f 3 |
		tr -d ',)'
}

module="$(head -1 go.mod | awk '{ print $2 }')"
version="$(_version)"

if [ "$version" = "" ]; then
	version="devel $(git log -n 1 --format='format: +%h %cd' HEAD)"
fi

default_tags="netgo osusergo"
default_ldflags=$(printf -- "-X '%s/version.Build=%s'" "$module" "$version")

if [ -z "$LDFLAGS" ]; then
	LDFLAGS="$default_ldflags"
else
	LDFLAGS="$LDFLAGS $default_ldflags"
fi

if [ -z "$TAGS" ]; then
	TAGS="$default_tags"
else
	TAGS="$TAGS $default_tags"
fi

for bin in $(grep -vE "^#" make.dep | awk '{ print $1 }'); do
	if ! hash "$bin" 2> /dev/null; then
		url=$(grep "^$bin" make.dep | awk '{ print $2 }')
		>&2 printf "missing binary %s\n" "$bin"
		>&2 printf "install binary via: %s\n" "$url"
		exit 1
	fi
done

yarn_() {
	bin="node_modules/.bin/lessc"

	if [ ! -f "$bin" ]; then
		yarn install
	fi

	"$bin" --clean-css static/less/main.less static/main.css
	"$bin" --clean-css static/less/auth.less static/auth.css
	"$bin" --clean-css static/less/error.less static/error.css
}

ui() {
	yarn_
	if [ -z "$1" ]; then
		find . -name template -type d -exec qtc -dir {} \;
	else
		dir="$1/template"

		if [ "$1" = "template" ]; then
			dir="template"
		fi

		if [ ! -d "$dir" ]; then
			>&2 printf "cannot find directory %s\n" "$dir"
			exit 1
		fi
		qtc -dir "$dir"
	fi
}

build() {
	[ ! -d bin ] && mkdir bin

	cmd="$1"

	if [ -z "$cmd" ]; then
		cmd="$(ls cmd)"
	fi

	go generate ./...

	for c in $cmd; do
		if [ ! -d cmd/"$c" ]; then
			>&2 printf "unknown package %s\n" "$c"
			exit 1
		fi
		set -x
		GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$LDFLAGS" \
			-tags "$TAGS" \
			-o bin/"$c" ./cmd/"$c"
		set +x
	done
}

help_() {
	case "$1" in
		ui)
			printf "compile the ui templates\n"
			printf "usage: make.sh ui [component]\n\n"
			printf "components:\n"
			components=$(find . -name template -type d | tr '/' ' ' | sort)
			components=$(echo "$components" | awk '{ print $2 }' | grep -v template)
			for c in $components; do
				printf "  %s\n" "$c"
			done
			;;
		build)
			printf "build one of the Go programs in cmd\n"
			printf "usage: make.sh build <dir>\n"
			;;
		runner)
			printf "compile the offline runner\n"
			printf "usage: make.sh runner\n"
			;;
		consumer)
			printf "compile the consumer\n"
			printf "usage: make.sh consumer\n"
			;;
		server)
			printf "compile the server\n"
			printf "usage: make.sh server\n"
			;;
		worker)
			printf "compile the worker\n"
			printf "usage: make.sh worker\n"
			;;
		clean)
			printf "remove built binaries\n"
			;;
		css)
			printf "compile the less to css\n"
			;;
		manif)
			printf "create a sum manifest file\n"
			;;
		*)
			printf "build the server and offline runner\n"
			printf "usage: make.sh [build|clean|css|manif|runner|server|ui|worker]\n"
			;;
	esac
}

manif() {
	cd bin

	[ -f sum.manif ] && rm sum.manif

	sha256sum djinn* | awk '{ print "SHA256 ("$2") = " $1 }' > sum.manif

	cd - > /dev/null
}

case "$1" in
	ui)
		shift 1
		ui "$1"
		;;
	build)
		shift 1
		build "$1"
		;;
	runner)
		build djinn
		;;
	consumer)
		build djinn-consumer
		;;
	server)
		build djinn-server
		;;
	worker)
		build djinn-worker
		;;
	scheduler)
		build djinn-scheduler
		;;
	curator)
		build djinn-curator
		;;
	help)
		shift 1
		help_ "$1"
		;;
	clean)
		rm -f bin/* djinn djinn-consumer djinn-curator djinn-scheduler djinn-server djinn-worker sum.manif
		find . -name "*.log" -exec rm -f {} \;
		go clean -cache -testcache
		;;
	manif)
		manif
		;;
	css)
		yarn_
		;;
	*)
		if [ "$1" = "" ]; then
			go test ./...
			ui
			build
			manif
		else
			>&2 printf "unknown job %s\n" "$1"
			exit 1
		fi
esac
