# Vibrant.uz — bosqichma-bosqich deploy

## Oldindan

- [ ] DNS Cloudflare: `stream.vibrant.uz` → server IP, **Proxied** (sariq bulut)
- [ ] `ingest.stream.vibrant.uz` → server IP, **DNS only** (kulrang)
- [ ] Cloudflare SSL/TLS: **Full** (origin sertifikat kerak bo'ladi)
- [ ] Certbot uchun: DNS **DNS only** (kulrang bulut) yoki Full + origin cert
- [ ] `sshpass` o'rnatilgan (lokal Mac/Linux)
- [ ] `for-deploy.txt` to'ldirilgan

```bash
cp for-deploy.txt.example for-deploy.txt
# IP, parol, domenlarni tahrirlang
```

---

## Bosqich 0 — Port tekshiruv (boshqa loyihalar bo'lsa)

Serverda:
```bash
FRONTEND_PORT=3002 GATEWAY_PORT=8080 bash scripts/check-server-ports.sh
```

Lokal (SSH orqali, `for-deploy.txt` kerak):
```bash
bash scripts/check-server-ports-remote.sh
```

Band port bo'lsa `for-deploy.txt` da `Frontend port` yoki `Gateway port` o'zgartiring.

| Port | Kim ishlatadi | O'zgartirish |
|------|---------------|--------------|
| 3002 | Next.js | `Frontend port` |
| 8080 | API gateway | `Gateway port` (keyinroq deploy) |
| 8090 | HLS | docker-compose |
| 8889 | WHIP | mediamtx |
| 1935 | RTMP | nginx-rtmp |

---

## Bosqich 1 — DNS tekshiruv

```bash
dig +short stream.vibrant.uz
# Server IP ni ko'rsatishi kerak
```

---

## Bosqich 2 — To'liq deploy (kod + docker + servislar + SSL)

Lokal mashinadan (loyiha ildizida):

```bash
# sshpass kerak: brew install sshpass  (Mac)
bash scripts/deploy.sh
```

Bu qiladi:
1. Go + frontend build (ikki domen URL bilan)
2. Serverga yuklash
3. Docker (Postgres, Redis, NATS, RTMP, MediaMTX, HLS)
4. Migratsiya
5. Go servislar + Next.js
6. Nginx + Let's Encrypt SSL

SSL xato bersa (Cloudflare proxy):
```bash
# Cloudflare da DNS-only qiling, keyin serverda:
SETUP_SSL=1 CERTBOT_EMAIL=siz@email.com bash /opt/sahiy-stream/scripts/setup-nginx-ssl.sh
```

---

## Bosqich 3 — Tekshiruv

```bash
curl -s https://stream.vibrant.uz/health
curl -sI https://stream.vibrant.uz/
```

Brauzer:
- https://stream.vibrant.uz — panel
- https://stream.vibrant.uz/studio/broadcast — kamera efir

---

## Bosqich 4 — Qayta deploy (kod yangilanganda)

```bash
bash scripts/deploy.sh
# SSL o'tkazib: SETUP_SSL=0 bash scripts/deploy.sh
```

---

## Cloudflare sozlamalari

| Sozlama | Qiymat |
|---------|--------|
| SSL/TLS | Full yoki Full (strict) |
| WebSockets | ON |
| HTTP/3 | ixtiyoriy |

Agar **Flexible** bo'lsa — origin HTTP, lekin kamera/WHIP uchun **Full** tavsiya.

---

## Muammolar

| Muammo | Yechim |
|--------|--------|
| `Couldn't find any pages or app directory` (serverda `npm run build`) | Serverda `src/` yo'q — **lokalda** `bash scripts/deploy.sh` yoki `git pull` + `bash scripts/build-frontend-server.sh` |
| certbot failed | DNS-only (kulrang bulut), 80-port ochiq |
| CORS xato | `GATEWAY_CORS_ORIGINS` da `https://stream.vibrant.uz` |
| Kamera ishlamaydi | HTTPS + `mediamtx.yml` da to'g'ri `__SERVER_IP__` |
| 502 Bad Gateway | `next start` port 3002, nginx `__FRONTEND_PORT__` |

---

## Fayl tuzilmasi

```
stream.vibrant.uz     → nginx → Next.js :3002
stream.vibrant.uz → nginx → frontend, gateway :8080, HLS :8090, WHIP :8889
```
