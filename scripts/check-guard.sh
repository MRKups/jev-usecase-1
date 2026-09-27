#!/bin/sh
set -eu

root=$(git rev-parse --show-toplevel)
cd "$root"

printf '==> Checking Go formatting (gofmt)...\n'
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    printf 'error: unformatted Go files detected:\n%s\n' "$unformatted" >&2
    printf 'Run "task fmt" or "gofmt -w ." to format.\n' >&2
    exit 1
fi

printf '==> Running static analysis (go vet)...\n'
go vet ./...

printf '==> Running host tests with race detector (go test -race)...\n'
go test -race ./...

printf '==> Verifying pure CGO_ENABLED=0 compilation...\n'
CGO_ENABLED=0 go build -o /dev/null ./cmd/ticket-eval

printf '==> Checking test guard (no unjustified test function deletions)...\n'
tmp=$(mktemp -d "${TMPDIR:-/tmp}/jev-guard.XXXXXX")
trap 'rm -rf "$tmp"' 0 INT TERM

# Count Test functions in HEAD vs worktree
if git rev-parse --verify --quiet HEAD > /dev/null; then
    git grep -E "^func Test" HEAD -- '*.go' | wc -l > "$tmp/before"
else
    echo "0" > "$tmp/before"
fi

grep -r -E "^func Test" internal/ cmd/ --include="*_test.go" 2>/dev/null | wc -l > "$tmp/after"

before=$(cat "$tmp/before" | tr -d ' ')
after=$(cat "$tmp/after" | tr -d ' ')

if [ "$before" -gt "$after" ]; then
    diff=$((before - after))
    printf 'error: %d net test function(s) removed (before: %d, after: %d).\n' "$diff" "$before" "$after" >&2
    printf 'Restore tests or justify their removal under rule DEV-13.\n' >&2
    exit 1
fi

printf '==> Guard checks passed successfully (%d tests verified).\n' "$after"
