#!/bin/bash
# UbuntuLink Android manual build (aapt2 flow).
# Order matters: zipalign BEFORE apksigner sign.
set -e
cd "$(dirname "$0")"

SDK="$HOME/Android/Sdk"
BT="$SDK/build-tools/35.0.0"
PLATFORM="$SDK/platforms/android-35"
APP_ID="com.ubuntulink.app"
VERSION="0.4.0"

# Debug keystore (local only, never commit).
KS="ubuntulink-debug.keystore"
if [ ! -f "$KS" ]; then
    keytool -genkeypair -keystore "$KS" -alias android -keyalg RSA -keysize 2048 \
        -validity 10950 -storepass android -keypass android \
        -dname "CN=UbuntuLink Debug, OU=Dev, O=Local" 2>/dev/null
fi

rm -rf build && mkdir -p build/gen build/obj build/apk

echo "== aapt2 compile =="
"$BT/aapt2" compile --dir app/src/main/res -o build/res.zip

echo "== aapt2 link =="
"$BT/aapt2" link -o build/apk/unsigned.apk \
    -I "$PLATFORM/android.jar" \
    --manifest app/src/main/AndroidManifest.xml \
    --java build/gen \
    build/res.zip

echo "== javac =="
find app/src/main/java -name "*.java" > build/sources.txt
find build/gen -name "*.java" >> build/sources.txt
javac -encoding UTF-8 -source 8 -target 8 \
    -cp "$PLATFORM/android.jar" \
    -d build/obj \
    @build/sources.txt > build/javac.log 2>&1 || true
if grep -q "error:" build/javac.log; then
    echo "JAVAC FAILED:"
    grep "error:" build/javac.log | head -20
    exit 1
fi
echo "javac OK"

echo "== d8 =="
"$BT/d8" --lib "$PLATFORM/android.jar" \
    --output build/apk \
    $(find build/obj -name "*.class")

echo "== add classes.dex =="
(cd build/apk && zip -q unsigned.apk classes.dex)

echo "== zipalign =="
"$BT/zipalign" -f 4 build/apk/unsigned.apk build/apk/aligned.apk

echo "== apksigner =="
"$BT/apksigner" sign --ks "$KS" --ks-pass pass:android \
    --out "build/apk/ubuntulink-v${VERSION}.apk" build/apk/aligned.apk

echo "== verify =="
"$BT/apksigner" verify "build/apk/ubuntulink-v${VERSION}.apk" && echo "APK_OK: build/apk/ubuntulink-v${VERSION}.apk"
ls -la "build/apk/ubuntulink-v${VERSION}.apk"
