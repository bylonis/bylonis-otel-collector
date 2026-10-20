#!/usr/bin/env bash
# Fails when a tracked file gains "signoz" references (case-insensitive) beyond
# the inherited ones recorded in scripts/signoz-refs-baseline.txt.
#
# Files listed in scripts/signoz-refs-allow.txt (one bash glob per line, '*'
# crosses '/', always followed by the reason after '#') are not counted: they
# have to name signoz (license, history, persisted names, the guard itself).
#
# The baseline only shrinks: removing references is always fine. After a
# deliberate change (a new compatibility alias, an upstream cherry-pick),
# regenerate it with:  scripts/check-signoz-refs.sh --update
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
baseline=scripts/signoz-refs-baseline.txt
allow=scripts/signoz-refs-allow.txt

mapfile -t allowed_globs < <(sed -e 's/#.*//' -e 's/[[:space:]]*$//' -e '/^$/d' "$allow")

is_allowed() {
	local pattern
	for pattern in "${allowed_globs[@]}"; do
		# shellcheck disable=SC2053 # the pattern is a glob on purpose
		[[ $1 == $pattern ]] && return 0
	done
	return 1
}

# "<count> <path>" for every tracked file that mentions signoz, sorted by path
current() {
	git grep -cil signoz -- . ':!scripts/signoz-refs-baseline.txt' |
		while read -r f; do
			is_allowed "$f" && continue
			printf '%s %s\n' "$(grep -oi signoz "$f" | wc -l | tr -d ' ')" "$f"
		done | sort -k2
}

if [[ ${1:-} == --update ]]; then
	current >"$baseline"
	echo "baseline updated: $(wc -l <"$baseline") files"
	exit 0
fi

fail=0
while read -r count file; do
	allowed=$(awk -v f="$file" '$2 == f { print $1 }' "$baseline")
	if [[ -z $allowed ]]; then
		echo "::error file=$file::$file: new file with $count signoz reference(s)"
		fail=1
	elif ((count > allowed)); then
		echo "::error file=$file::$file: signoz references grew from $allowed to $count"
		fail=1
	fi
done < <(current)

if ((fail)); then
	echo "Use bylonis names. If the reference is intended, run scripts/check-signoz-refs.sh --update."
	exit 1
fi
echo "ok: no new signoz references"
