echo "Download centrifugo"
wget --timeout=10 https://github.com/centrifugal/centrifugo/releases/download/v4.0.3/centrifugo_4.0.3_linux_amd64.tar.gz
tar xvfz centrifugo_4.0.3_linux_amd64.tar.gz centrifugo
rm -rf centrifugo_4.0.3_linux_amd64.tar.gz
chmod +x centrifugo

echo "Download traefik"
wget --timeout=10 https://github.com/traefik/traefik/releases/download/v2.9.6/traefik_v2.9.6_linux_amd64.tar.gz
tar xvfz traefik_v2.9.6_linux_amd64.tar.gz traefik
rm -rf traefik_v2.9.6_linux_amd64.tar.gz
chmod +x traefik
