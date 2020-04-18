#!/bin/sh

set -e

TAGS="netgo osusergo"
LFLAGS="-ldflags \"-X=main.Build=$(git rev-parse HEAD)\""

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
	yarn_
}

build() {
	cmd="$1"

	if [ -z "$cmd" ]; then
		cmd="$(ls cmd)"
	fi

	go generate ./...

	for c in "$cmd"; do
		if [ ! -d cmd/"$c" ]; then
			>&2 printf "unknown package %s\n" "$c"
			exit 1
		fi
		set -x
		GOOS="$GOOS" GOARCH="$GOARCH" go build $LFLAGS -tags "$TAGS" -o "$c".out ./cmd/"$c"
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
		*)
			printf "build the server and offline runner\n"
			printf "usage: make.sh [ui|runner|server|worker|clean]\n"
			;;
	esac
}

case "$1" in
	ui)
		shift 1
		ui "$1"
		;;
	runner)
		build thrall
		;;
	server)
		build thrall-server
		;;
	worker)
		build thrall-worker
		;;
	help)
		shift 1
		help_ "$1"
		;;
	clean)
		rm -f *.out
		rm -f *.tar
		go clean -x -testcache
		;;
	*)
		if [ "$1" = "" ]; then
			go test -cover ./...
			ui
			build
		else
			>&2 printf "unknown job %s\n" "$1"
			exit 1
		fi
esac