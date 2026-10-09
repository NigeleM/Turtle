#!/bin/sh
# Removes Turtle: sudo /usr/local/share/turtle/uninstall.sh
set -e
if [ "$(id -u)" != 0 ]; then
	echo "Run it with sudo: sudo $0"
	exit 1
fi
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -u /Applications/Turtle.app 2>/dev/null || true
rm -rf /Applications/Turtle.app /usr/local/bin/turtle /usr/local/share/turtle
pkgutil --forget com.genesys.turtle >/dev/null 2>&1 || true
pkgutil --forget com.genesys.turtle.app >/dev/null 2>&1 || true
echo "Turtle is removed."
