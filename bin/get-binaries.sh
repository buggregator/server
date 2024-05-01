echo "Download centrifugo"
#wget --timeout=10 https://github.com/centrifugal/centrifugo/releases/download/v4.0.3/centrifugo_4.0.3_linux_amd64.tar.gz
#tar xvfz centrifugo_4.0.3_linux_amd64.tar.gz centrifugo
#rm -rf centrifugo_4.0.3_linux_amd64.tar.gz
#chmod +x centrifugo

echo "Download dolt db"
wget --timeout=10 https://github.com/dolthub/dolt/releases/download/v1.35.12/dolt-linux-amd64.tar.gz
tar xvfz dolt-linux-amd64.tar.gz dolt-linux-amd64/bin/dolt
rm -rf dolt-linux-amd64.tar.gz
mv dolt-linux-amd64/bin/dolt dolt
rm -rf dolt-linux-amd64
chmod +x dolt
