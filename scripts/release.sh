#!/bin/sh
# Tags the next version on main: scripts/release.sh patch|minor|major (or make release-patch etc.).
# Git tags are the only place the version lives; builds take it from `git describe`. Pushing stays a
# separate step, the script prints the command.
set -eu

part=${1:-}
case "$part" in
patch | minor | major) ;;
*)
	echo "Usage: $0 patch|minor|major" >&2
	exit 2
	;;
esac

if [ "$(git rev-parse --abbrev-ref HEAD)" != main ]; then
	echo "Releases are tagged on main" >&2
	exit 1
fi
if [ -n "$(git status --porcelain)" ]; then
	echo "Commit or stash your changes first" >&2
	exit 1
fi

current=$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' 2>/dev/null || echo v0.0.0)
if [ "$current" != v0.0.0 ] && [ "$(git rev-list -n 1 "$current")" = "$(git rev-parse HEAD)" ]; then
	echo "This commit is already $current" >&2
	exit 1
fi

# shellcheck disable=SC2046 # split "1.2.3" into its parts
set -- $(echo "${current#v}" | tr . ' ')
case "$part" in
major) next="v$(($1 + 1)).0.0" ;;
minor) next="v$1.$(($2 + 1)).0" ;;
patch) next="v$1.$2.$(($3 + 1))" ;;
esac

git tag -a "$next" -m "ragbag $next"
echo "Tagged $next (was $current). Publish it with: git push origin main $next"
