#! /bin/bash
set -e

function print_help {
	printf "Available Commands:\n";
	printf "  run\n"
}

function run_run {
  go get github.com/advanderveer/rpctocli
  cd example
  rpctocli
}

case $1 in
	"run") run_run ;;
	*) print_help ;;
esac
