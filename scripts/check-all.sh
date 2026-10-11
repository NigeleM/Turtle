#!/bin/sh
# check-all.sh: the checks CI runs, on this computer, before a push.
#
#   scripts/check-all.sh          macOS (or this system), Windows build, Linux in Docker,
#                                 and bench/ against origin/main
#   scripts/check-all.sh --quick  skip Docker
#
# It tests what's committed (HEAD), not the working folder, so a new file
# that was never added fails here instead of on GitHub. Commit first.
set -eu

PKGS="./legacy/Files/... ./ast/... ./cmd/... ./docs/... ./evaluator/... ./lexer/... ./object/... ./parser/... ./postgres/... ./mysql/... ./sqlite/... ./toml/... ./token/... ./syntax/... ./format/... ./lsp/... ./repl/..."
QUICK=0
[ "${1:-}" = "--quick" ] && QUICK=1

cd "$(git rev-parse --show-toplevel)"
repo=$(pwd)
fail=0
step() { printf '\n== %s\n' "$1"; }

step "files git doesn't have"
missing=$(git ls-files --others --exclude-standard -- '*.go' '*.turtle' '*.trt' 'go.mod')
ignored=$(git ls-files --others --ignored --exclude-standard -- '*.go' '*.turtle' | grep -v '^testdata/sql/work' || true)
if [ -n "$missing$ignored" ]; then
	echo "not committed:"; echo "$missing$ignored" | sed '/^$/d; s/^/  /'
	fail=1
else
	echo "none"
fi
if ! git diff --quiet HEAD; then
	echo "(you have uncommitted changes; they aren't part of this check)"
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
git archive HEAD | tar -x -C "$work"
cd "$work"

step "$(go env GOOS): vet and test"
# shellcheck disable=SC2086
if go vet $PKGS && TURTLE_REQUIRE_SQLITE3=1 go test $PKGS >"$work/test.log" 2>&1; then
	echo "ok"
else
	grep -v '^ok\|no test files' "$work/test.log" || true
	fail=1
fi

step "windows: vet and build every package and test"
# shellcheck disable=SC2086
if GOOS=windows go vet $PKGS; then
	for p in $(go list $PKGS); do
		GOOS=windows go test -c -o /dev/null "$p" >/dev/null || fail=1
	done
	GOOS=windows go build -o /dev/null ./cmd/turtle || fail=1
	echo "done"
else
	fail=1
fi

if [ "$QUICK" = 0 ]; then
	if docker info >/dev/null 2>&1; then
		for v in 1.24 latest; do
			step "linux, Go $v (Docker)"
			docker pull -q "golang:$v" >/dev/null || fail=1
			out=$(docker run --rm -v "$work":/src:ro "golang:$v" sh -c "
				apt-get update -qq >/dev/null 2>&1 && apt-get install -y -qq sqlite3 >/dev/null 2>&1
				cp -r /src /w && cd /w && TURTLE_REQUIRE_SQLITE3=1 go test $PKGS 2>&1 | grep -v '^ok\\|no test files'
				exit 0" 2>&1) || fail=1
			if [ -n "$out" ]; then
				echo "$out"
				fail=1
			else
				echo "ok"
			fi
		done
	else
		step "linux: skipped, Docker isn't running"
	fi
fi

step "performance against origin/main (no slower, no bigger)"
if (cd "$repo" && go run scripts/perfcheck.go -base origin/main); then :; else fail=1; fi

echo
if [ "$fail" = 0 ]; then
	echo "all checks passed"
else
	echo "some checks FAILED (see above)"
	exit 1
fi
