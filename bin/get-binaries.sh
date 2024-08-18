OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if ! command -v wget &> /dev/null
then
    echo "wget could not be found. Please install wget and try again."
    exit 1
fi

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH="arm64"
else
    echo "Failed to download binaries unsupported architecture: $ARCH"
    exit 1
fi

echo "Detected OS: $OS, Architecture: $ARCH"

echo "Downloading Centrifugo"
wget --timeout=10 "https://github.com/centrifugal/centrifugo/releases/download/v4.0.3/centrifugo_4.0.3_${OS}_${ARCH}.tar.gz"
tar xvfz centrifugo_4.0.3_${OS}_${ARCH}.tar.gz centrifugo
rm -rf centrifugo_4.0.3_${OS}_${ARCH}.tar.gz
chmod +x centrifugo

echo "Downloading Dolt"
wget --timeout=10 "https://github.com/dolthub/dolt/releases/download/v1.42.8/dolt-$OS-$ARCH.tar.gz"
tar xvfz dolt-$OS-$ARCH.tar.gz dolt-$OS-$ARCH/bin/dolt
rm -rf dolt-$OS-$ARCH.tar.gz
mv dolt-$OS-$ARCH/bin/dolt dolt
rm -rf dolt-$OS-$ARCH
chmod +x dolt
