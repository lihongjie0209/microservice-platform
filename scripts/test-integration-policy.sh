#!/bin/sh
set -eu

check_project() {
	project=$1
	local_command=$(make -s -n -C "$project" test-integration)
	ci_command=$(make -s -n -C "$project" ci-test-integration)

	case "$local_command" in
		*"-tags=integration"*"-run '^$'"*) ;;
		*) echo "$project: test-integration must compile without running containers" >&2; return 1 ;;
	esac
	case "$ci_command" in
		*"-tags=integration"*"-count=1"*) ;;
		*) echo "$project: ci-test-integration must execute the integration suite" >&2; return 1 ;;
	esac
	if ! grep -R -q 'run: make ci-test-integration' "$project/.github/workflows"; then
		echo "$project: GitHub Actions must call make ci-test-integration" >&2
		return 1
	fi
}

check_project libraries/platform-go
for project in services/*-service; do
	check_project "$project"
done
