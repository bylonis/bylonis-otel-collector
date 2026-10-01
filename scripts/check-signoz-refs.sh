#!/usr/bin/env bash
# Fails when a tracked file gains "signoz" references (case-insensitive) beyond
# the inherited ones recorded in scripts/signoz-refs-baseline.txt.
#
# The baseline only shrinks: removing references is always fine. After a
# deliberate change (a new compatibility alias, an upstream cherry-pick),
# regenerate it with:  scripts/check-signoz-refs.sh --update
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
baseline=scripts/signoz-refs-baseline.txt

# "<count> <path>" for every tracked file that mentions signoz, sorted by path
current() {
	git grep -cil signoz -- . ':!scripts/signoz-refs-baseline.txt' |
		while read -r f; do
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
		echo "::error file=$file::new file with $count signoz reference(s)"
		fail=1
	elif ((count > allowed)); then
		echo "::error file=$file::signoz references grew from $allowed to $count"
		fail=1
	fi
done < <(current)

if ((fail)); then
	echo "Use bylonis names. If the reference is intended, run scripts/check-signoz-refs.sh --update."
	exit 1
fi
echo "ok: no new signoz references"
