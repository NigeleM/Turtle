#!/bin/sh
# Builds turtle.pkg: turtle (Apple silicon and Intel) in /usr/local/bin,
# Turtle.app in /Applications (.turtle files get its icon and run on a
# double-click), and /usr/local/share/turtle (turtle-run, uninstall.sh,
# LICENSE, NOTICE).
#
#   installer/macos/build-pkg.sh <version> [output folder]
#
# The icons are installer/icons: turtle-app.png (1024px, Turtle app and
# its applet icon) and turtle-file.icns (.turtle files).
set -eu
version=${1:?usage: build-pkg.sh <version> [output folder]}
version=${version#v}
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../.." && pwd)
out=$(mkdir -p "${2:-$repo/release}" && cd "${2:-$repo/release}" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# One binary for both kinds of Mac.
cd "$repo"
for arch in arm64 amd64; do
	CGO_ENABLED=0 GOOS=darwin GOARCH=$arch go build -ldflags "-s -w -X main.version=v$version" -o "$work/turtle-$arch" ./cmd/turtle
done
# Two parts, each installed into a folder that's already there, so the
# folders themselves (/usr/local, /Applications) are left as they are.
cli="$work/cli"
mkdir -p "$cli/bin" "$cli/share/turtle" "$work/apps"
lipo -create -output "$cli/bin/turtle" "$work/turtle-arm64" "$work/turtle-amd64"
chmod 755 "$cli/bin/turtle"
cp "$here/turtle-run" "$here/uninstall.sh" "$cli/share/turtle/"
cp LICENSE NOTICE "$cli/share/turtle/"

# Turtle.app: an applet that hands Finder's double-click to turtle-run.
app="$work/apps/Turtle.app"
osacompile -o "$app" "$here/Turtle.applescript" 2>/dev/null
iconset="$work/app.iconset"
mkdir -p "$iconset"
for n in 16 32 128 256 512; do
	sips -z $n $n installer/icons/turtle-app.png --out "$iconset/icon_${n}x${n}.png" >/dev/null
	sips -z $((n * 2)) $((n * 2)) installer/icons/turtle-app.png --out "$iconset/icon_${n}x${n}@2x.png" >/dev/null
done
plist="$app/Contents/Info.plist"
icon=$(plutil -extract CFBundleIconFile raw "$plist") # droplet, for an app that takes files
iconutil -c icns "$iconset" -o "$app/Contents/Resources/${icon%.icns}.icns"
cp installer/icons/turtle-file.icns "$app/Contents/Resources/turtle-file.icns"
rm -f "$app/Contents/Resources/Assets.car" # the applet's own icon, which would win
plutil -replace CFBundleIdentifier -string "com.genesys.turtle.app" "$plist"
plutil -replace CFBundleName -string "Turtle" "$plist"
plutil -replace CFBundleShortVersionString -string "$version" "$plist"
plutil -replace CFBundleVersion -string "$version" "$plist"
plutil -replace NSHumanReadableCopyright -string "Copyright 2017-2026 Nigele McCoy" "$plist"
plutil -remove CFBundleIconName "$plist" 2>/dev/null || true
plutil -replace UTExportedTypeDeclarations -json '[{
	"UTTypeIdentifier": "com.genesys.turtle.script",
	"UTTypeDescription": "Turtle program",
	"UTTypeConformsTo": ["public.script", "public.plain-text"],
	"UTTypeIconFile": "turtle-file.icns",
	"UTTypeTagSpecification": {"public.filename-extension": ["turtle"]}
}]' "$plist"
plutil -replace UTImportedTypeDeclarations -json '[{
	"UTTypeIdentifier": "com.genesys.turtle.trt",
	"UTTypeDescription": "Turtle program (.trt)",
	"UTTypeConformsTo": ["public.plain-text"],
	"UTTypeTagSpecification": {"public.filename-extension": ["trt"]}
}]' "$plist"
plutil -replace CFBundleDocumentTypes -json '[{
	"CFBundleTypeName": "Turtle program",
	"CFBundleTypeRole": "Viewer",
	"CFBundleTypeIconFile": "turtle-file.icns",
	"LSHandlerRank": "Owner",
	"LSItemContentTypes": ["com.genesys.turtle.script"]
}, {
	"CFBundleTypeName": "Turtle program (.trt)",
	"CFBundleTypeRole": "Viewer",
	"LSHandlerRank": "Alternate",
	"LSItemContentTypes": ["com.genesys.turtle.trt"]
}]' "$plist"
plutil -lint "$plist" >/dev/null
# Signing. With TURTLE_SIGN_APP (a "Developer ID Application: ..." name),
# TURTLE_SIGN_PKG ("Developer ID Installer: ...") and TURTLE_NOTARY_PROFILE
# (a notarytool keychain profile) set, the package is signed, notarized and
# stapled, so Gatekeeper accepts it. Without them it's signed ad hoc, as
# before: the edits above break the applet's seal, so it's signed again.
sign_app=${TURTLE_SIGN_APP:-}
sign_pkg=${TURTLE_SIGN_PKG:-}
notary=${TURTLE_NOTARY_PROFILE:-}
if [ -n "$sign_app" ]; then
	codesign --force --options runtime --timestamp --sign "$sign_app" "$cli/bin/turtle"
	codesign --force --options runtime --timestamp --sign "$sign_app" "$app"
	codesign --verify --strict --deep "$app"
else
	codesign --force --sign - "$app"
fi

# The app installs where it's built to go, not "relocated" to another copy.
pkgbuild --analyze --root "$work/apps" "$work/components.plist" >/dev/null
plutil -replace 0.BundleIsRelocatable -bool NO "$work/components.plist"
pkgbuild --root "$work/apps" --component-plist "$work/components.plist" \
	--scripts "$here/scripts" --identifier com.genesys.turtle.app \
	--version "$version" --install-location /Applications \
	"$work/app.pkg" >/dev/null
pkgbuild --root "$cli" --identifier com.genesys.turtle \
	--version "$version" --install-location /usr/local --ownership recommended \
	"$work/cli.pkg" >/dev/null
# The installer's licence page shows NOTICE and then the licence (LICENSE,
# Apache 2.0), and installs only if the user agrees.
res="$work/resources"
mkdir -p "$res"
# One licence page: NOTICE first, then the Apache licence.
installer/license.sh "$res/LICENSE.txt"
productbuild --synthesize --package "$work/cli.pkg" --package "$work/app.pkg" "$work/distribution.xml"
sed -i '' '/<installer-gui-script/a\
    <license file="LICENSE.txt" mime-type="text/plain"/>
' "$work/distribution.xml"
xmllint --noout "$work/distribution.xml"
productbuild --distribution "$work/distribution.xml" --package-path "$work" --resources "$res" "$work/turtle.pkg" >/dev/null
if [ -n "$sign_pkg" ]; then
	productsign --sign "$sign_pkg" --timestamp "$work/turtle.pkg" "$out/turtle.pkg"
	if [ -n "$notary" ]; then
		xcrun notarytool submit "$out/turtle.pkg" --keychain-profile "$notary" --wait
		xcrun stapler staple "$out/turtle.pkg"
		xcrun stapler validate "$out/turtle.pkg"
	fi
	pkgutil --check-signature "$out/turtle.pkg" | head -4
else
	mv "$work/turtle.pkg" "$out/turtle.pkg"
fi
echo "built $out/turtle.pkg"
