#!/bin/sh

for bin in $(grep -vE "^#" make.dep | awk '{ print $1 }'); do
	if ! hash "$bin" 2> /dev/null; then
		url=$(grep "^$bin" make.dep | awk '{ print $2 }')
		>&2 printf "missing binary %s\n" "$bin"
		>&2 printf "install binary via: %s\n" "$url"
		exit 1
	fi
done

_err() {
	>&2 echo $@
	exit 1
}

_version() {
	tag=$(
		git log --decorate=full --format=format:%d |
			head -1 |
			tr ',' '\n' |
			grep tag: |
			cut -d / -f 3 |
			tr -d ',)'
	)

	if [ -z "$tag" ]; then
		echo "devel $(git log -n 1 --format='format:%h %cd' HEAD)"
	else
		echo "$tag"
	fi
}

lessc_bin="node_modules/.bin/lessc"

LDFLAGS="$LDFLAGS $(printf -- "-X 'djinn-ci.com/version.Build=%s'" "$(_version)")"
TAGS="$TAGS netgo osusergo"

_exec() {
	set -x
	"$@"
	set +x
}

_css() {
	_exec "$lessc_bin" --clean-css template/static/less/"$1".less template/static/"$1".css
}

_ui() {
	[ ! -f "$lessc_bin" ] && _exec yarn install

	_css main
	_css auth
	_css error

	_exec qtc
}

_build() {
	[ ! -d bin ] && mkdir bin

	targets="$1"

	if [ -z "$targets" ]; then
		targets="$(ls cmd)"
	fi

	_exec go generate ./...

	for target in $targets; do
		GOOS="$GOOS" GOARCH="$GOARCH" _exec go build \
			-trimpath \
			-ldflags "$LDFLAGS" \
			-tags "$TAGS" \
			-o bin/"$target" \
			./cmd/"$target"
	done
}

_manif() {
	cd bin && {
		_exec sha256sum djinn* | awk '{ print "SHA256 ("$2") = " $1 }' > sum.manif
		cd - > /dev/null
	}
}

_test() {
	_exec go generate ./...
	_exec go test ./...
}

_clean() {
	_exec rm -f bin/*
	_exec find . -name "*.log" -exec rm -f {} \;
	_exec go clean -cache -testcache
}

case "$1" in
	build)
		_build
		;;
	clean)
		_clean
		;;
	css)
		_css main
		_css auth
		_css error
		;;
	ui)
		_ui
		;;
	*)
		if [ "$1" = "" ]; then
			_test
			_ui
			_build
			_manif
		elif [ -d "cmd/djinn-$1" ]; then
			_build "djinn-$1"
		elif [ "$1" = "runner" ]; then
			_build djinn-runner
		else
			_err "unknown target: $1"
		fi
esac
