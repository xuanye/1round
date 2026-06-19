# Deploy OneRound in a Private VPC

## Docker Compose

```bash
cd deploy
cp ../apps/server/config.example.yaml config.yaml
docker compose up -d --build
```

Keep the SQLite file on a persistent volume. Do not bake database files into images.

## systemd

```bash
sudo useradd --system --home /var/lib/oneround oneround
sudo mkdir -p /opt/oneround /var/lib/oneround
sudo chown -R oneround:oneround /var/lib/oneround
sudo cp oneround-server config.yaml /opt/oneround/
sudo cp systemd/oneround.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now oneround
```

## Nginx

Use `nginx.conf` as the HTTPS/WSS reverse proxy baseline. The Mini Program production domain must be registered in the WeChat allowed domain settings.
