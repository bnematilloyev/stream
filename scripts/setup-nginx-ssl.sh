#!/usr/bin/env bash
# Serverda nginx + SSL (Let's Encrypt).
# Default: bitta domen — stream.vibrant.uz (frontend + API + HLS, Cloudflare CDN).
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
FRONTEND_PORT="${FRONTEND_PORT:-3002}"
GATEWAY_PORT="${GATEWAY_PORT:-8080}"
HLS_PORT="${HLS_PORT:-8090}"
FRONTEND_DOMAIN="${FRONTEND_DOMAIN:-stream.vibrant.uz}"
LEGACY_API_DOMAIN="${LEGACY_API_DOMAIN:-api.stream.vibrant.uz}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-admin@vibrant.uz}"

resolve_cert_name() {
  if [[ -f "/etc/letsencrypt/live/${FRONTEND_DOMAIN}/fullchain.pem" ]]; then
    echo "${FRONTEND_DOMAIN}"
  elif [[ -f "/etc/letsencrypt/live/api.stream.vibrant.uz/fullchain.pem" ]]; then
    echo "api.stream.vibrant.uz"
  else
    echo "${FRONTEND_DOMAIN}"
  fi
}

CERT_NAME="$(resolve_cert_name)"

echo "==> Nginx va certbot..."
echo "    Domen: ${FRONTEND_DOMAIN} (+ ${LEGACY_API_DOMAIN} redirect)"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq nginx certbot python3-certbot-nginx

mkdir -p /var/www/certbot

install_http_only() {
  for domain in "$@"; do
    cat >"/etc/nginx/sites-available/${domain}" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${domain};

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 200 'ok';
        add_header Content-Type text/plain;
    }
}
EOF
    ln -sf "/etc/nginx/sites-available/${domain}" "/etc/nginx/sites-enabled/${domain}"
  done
  rm -f /etc/nginx/sites-enabled/default
  nginx -t && systemctl restart nginx
}

CERT_DIR="/etc/letsencrypt/live/${CERT_NAME}"
if [[ -f "${CERT_DIR}/fullchain.pem" && -f "${CERT_DIR}/privkey.pem" ]]; then
  echo "==> SSL sertifikat mavjud (${CERT_NAME}), certbot o'tkazildi"
else
  echo "==> HTTP-only (certbot uchun)..."
  install_http_only "${FRONTEND_DOMAIN}" "${LEGACY_API_DOMAIN}"

  echo "==> SSL sertifikat..."
  echo "    Cloudflare: DNS-only (kulrang) yoki SSL=Full + 80-port ochiq"
  certbot certonly --webroot -w /var/www/certbot \
    --non-interactive --agree-tos -m "${CERTBOT_EMAIL}" \
    -d "${FRONTEND_DOMAIN}" -d "${LEGACY_API_DOMAIN}" \
    --cert-name "${CERT_NAME}"
fi

echo "==> To'liq nginx konfiguratsiyasi..."
install -m 644 "${REMOTE_DIR}/infra/nginx/stream.vibrant.uz.conf" \
  "/etc/nginx/sites-available/${FRONTEND_DOMAIN}"

sed -i "s/__FRONTEND_PORT__/${FRONTEND_PORT}/g" \
  "/etc/nginx/sites-available/${FRONTEND_DOMAIN}"
sed -i "s/__GATEWAY_PORT__/${GATEWAY_PORT}/g" \
  "/etc/nginx/sites-available/${FRONTEND_DOMAIN}"
sed -i "s/__HLS_PORT__/${HLS_PORT}/g" \
  "/etc/nginx/sites-available/${FRONTEND_DOMAIN}"
sed -i "s|/etc/letsencrypt/live/stream.vibrant.uz/|/etc/letsencrypt/live/${CERT_NAME}/|g" \
  "/etc/nginx/sites-available/${FRONTEND_DOMAIN}"

ln -sf "/etc/nginx/sites-available/${FRONTEND_DOMAIN}" "/etc/nginx/sites-enabled/${FRONTEND_DOMAIN}"

install -m 644 "${REMOTE_DIR}/infra/nginx/api.stream.vibrant.uz.conf" \
  "/etc/nginx/sites-available/${LEGACY_API_DOMAIN}"
sed -i "s|/etc/letsencrypt/live/stream.vibrant.uz/|/etc/letsencrypt/live/${CERT_NAME}/|g" \
  "/etc/nginx/sites-available/${LEGACY_API_DOMAIN}"
ln -sf "/etc/nginx/sites-available/${LEGACY_API_DOMAIN}" \
  "/etc/nginx/sites-enabled/${LEGACY_API_DOMAIN}"

nginx -t && systemctl reload nginx

echo "OK: https://${FRONTEND_DOMAIN}/"
echo "OK: https://${FRONTEND_DOMAIN}/health"
