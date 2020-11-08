#!/bin/sh

set -e

LIBS="netgo osusergo"
BUILD="-X=version.Ref=$(git rev-parse HEAD)"

if [ -z "$LDFLAGS" ]; then
	LDFLAGS="$BUILD"
else
	LDFLAGS="$LDFLAGS $BUILD"
fi

if [ -z "$TAGS" ]; then
	TAGS="$LIBS"
else
	TAGS="$TAGS $LIBS"
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
		GOOS="$GOOS" GOARCH="$GOARCH" go build -gcflags "-e" \
			-ldflags "$LDFLAGS" \
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
		runner)
			printf "compile the offline runner\n"
			printf "usage: make.sh runner\n"
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
		dev)
			printf "copy the configuration files in dist/ for local dev\n"
			;;
		*)
			printf "build the server and offline runner\n"
			printf "usage: make.sh [clean|css|dev|manif|runner|server|ui|worker]\n"
			;;
	esac
}

manif() {
	shasum -a 256 bin/* | awk '{ print "SHA256 ("$2") = " $1 }' > sum.manif
}

case "$1" in
	ui)
		shift 1
		ui "$1"
		;;
	runner)
		build djinn
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
		rm -f bin/*
		find . -name "*.log" -exec rm -f {} \;
		go clean -x -testcache
		;;
	manif)
		manif
		;;
	css)
		yarn_
		;;
	dev)
		cp dist/*.toml .
		sed -i "s/\/var\/lib\/ssl\/server\.crt//g" *.toml
		sed -i "s/\/var\/lib\/ssl\/server\.key//g" *.toml
		sed -i "s/\/var\/lib\/djinn\/images/\/tmp/g" *.toml
		sed -i "s/\/var\/lib\/djinn\/artifacts/\/tmp/g" *.toml
		sed -i "s/\/var\/lib\/djinn\/objects/\/tmp/g" *.toml
		sed -i "s/\/var\/log\/djinn\/server\.log/\/dev\/stdout/g" *.toml
		sed -i "s/\/var\/log\/djinn\/worker\.log/\/dev\/stdout/g" *.toml
		sed -i "s/https/http/g" *.toml
		;;
	*)
		if [ "$1" = "" ]; then
			go test -gcflags "-e" -tags "$TAGS" -cover ./...
			ui
			build
			manif
		else
			>&2 printf "unknown job %s\n" "$1"
			exit 1
		fi
esac
