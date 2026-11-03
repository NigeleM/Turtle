#!/bin/sh
# Writes the licence page the installers show: NOTICE (copyright, authorship
# and the book, docs, logo and icons terms), then the Apache licence (LICENSE).
#
#   installer/license.sh <output file>
#
# Run from the repo root (the macOS and Windows builds both call it).
set -eu
out=${1:?usage: license.sh <output file>}
{
	cat NOTICE
	printf '\n\n=====================================================================\n\n'
	printf 'Apache License, Version 2.0 (the code of Turtle)\n\n'
	cat LICENSE
} >"$out"
