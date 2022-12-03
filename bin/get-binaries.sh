echo "Download centrifugo"
wget --timeout=10 https://github.com/centrifugal/centrifugo/releases/download/v4.0.3/centrifugo_4.0.3_linux_amd64.tar.gz
tar xvfz centrifugo_4.0.3_linux_amd64.tar.gz centrifugo
rm -rf centrifugo_4.0.3_linux_amd64.tar.gz
chmod +x centrifugo
