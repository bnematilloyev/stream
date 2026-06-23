# Seller efir boshlash qo'llanmasi (React + Flutter)

Bu hujjat seller (sotuvchi) **kamera/ekranni jonli efirga uzatishni** qanday boshlashini tushuntiradi:
React (web) va Flutter (mobil) uchun amaliy kod bilan.

---

## 1. Ikki yo'l: WHIP vs RTMP/SRT

Session javobida ikkala usul uchun ham ma'lumot keladi:

```jsonc
{
  "access_token": "eyJ...",          // Sahiy-Stream JWT (publish + featured uchun)
  "refresh_token": "eyJ...",
  "expires_at_unix": 1782292913,
  "channel_slug": "shop-273",
  "shop_id": 273,
  "whip_base_url": "https://stream.vibrant.uz",   // ← WHIP (brauzer/ilova kamerasi)
  "ingest": {
    "key_prefix": "sk_live_xxxxxxxxx",
    "rtmp_url": "rtmp://38.242.216.233:1935/live", // ← OBS / tashqi encoder
    "srt_url":  "srt://ingest.sahiy.stream:9000",  // ← OBS / SRT encoder
    "stream_key": "sk_live_xxxxxxxxxxxxxxxx"
  }
}
```

| Usul | Kim uchun | OBS kerakmi | Texnologiya |
|------|-----------|-------------|-------------|
| **WHIP** (WebRTC) | Oddiy seller — telefon/kompyuter kamerasidan to'g'ridan-to'g'ri | ❌ Yo'q | `whip_base_url` + `streamId` |
| **RTMP** | Professional — OBS, overlaylar, ekran ulashish | ✅ Ha | `rtmp_url` + `stream_key` |
| **SRT** | Past-kechikishli tashqi encoder | ✅ Ha | `srt_url` + `stream_key` |

> **Asosiy holat:** ko'pchilik seller **WHIP** ishlatadi — OBS kerak emas, ilova/brauzer ichidayoq kamera yoqib efirga chiqadi. RTMP/SRT kalitlari faqat "ilg'or" sellerlar uchun zaxira.

---

## 2. To'liq oqim (har ikki platforma uchun bir xil)

```
1. POST {market}/api/v1/seller/broadcast/session
      → access_token, whip_base_url, ingest{...}, channel_slug

2. POST {market}/api/v1/seller/broadcast/streams/
      { title, latency_mode:"ultra-low", ingest_protocol:"whip" }
      → { id: "<streamId UUID>" }

3. Kamerani publish qilish:
      WHIP:  {whip_base_url}/{streamId}/whip   ← WebRTC ulanish
      RTMP:  {rtmp_url}  +  stream_key         ← OBS sozlamasi

4. POST {market}/api/v1/seller/broadcast/streams/{streamId}/start
      → status: "live"

5. (ixtiyoriy) Mahsulot ko'rsatish — live-shopping:
      POST {whip_base_url}/v1/chat/{streamId}/featured   (Bearer access_token)

6. POST {market}/api/v1/seller/broadcast/streams/{streamId}/end
      → status: "ended"
```

> **Eslatma — domenlar:** 1, 2, 4, 6-qadamlar **market** backend'iga (`/api/v1/seller/...`).
> 3 va 5-qadamlar to'g'ridan-to'g'ri **Sahiy-Stream**'ga (`whip_base_url` = `https://stream.vibrant.uz`).

### WHIP publish tartibi muhim

Avval **WHIP ulanishni o'rnating** (kamera oqimi ketishni boshlasin), **keyin** `start` chaqiring.
Aks holda `start` "live" qiladi-yu, lekin video kelmaydi.

---

## 3. React (web) — WHIP orqali

Loyihada tayyor kutubxona ishlatiladi: [`@eyevinn/whip-web-client`](https://www.npmjs.com/package/@eyevinn/whip-web-client).
To'liq misol: `frontend/src/components/broadcast/CameraBroadcast.tsx`.

```bash
npm i @eyevinn/whip-web-client
```

```tsx
import { WHIPClient } from "@eyevinn/whip-web-client";

// 1. Kamera + mikrofon
const media = await navigator.mediaDevices.getUserMedia({
  video: { facingMode: "user", width: { ideal: 1280 }, height: { ideal: 720 }, frameRate: { ideal: 30 } },
  audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
});
videoRef.current.srcObject = media; // preview

// 2. WHIP endpoint:  {whip_base_url}/{streamId}/whip
const endpoint = `${session.whip_base_url}/${streamId}/whip`;

const client = new WHIPClient({
  endpoint,
  opts: {
    iceServers: [
      { urls: "stun:stun.l.google.com:19302" },
      { urls: "stun:stun1.l.google.com:19302" },
    ],
    iceGatheringTimeout: 3000,
  },
});

// 3. Avval ingest (oqim ketsin), keyin start
await client.ingest(media);
await startStream(streamId);   // market: POST /seller/broadcast/streams/{id}/start

// Tugatish:
await client.destroy();
media.getTracks().forEach((t) => t.stop());
await endStream(streamId);      // market: POST /seller/broadcast/streams/{id}/end
```

> ⚠️ **HTTPS shart.** `getUserMedia` faqat xavfsiz kontekstda (HTTPS yoki `localhost`) ishlaydi.
> Mobil brauzerda test qilsangiz, sahifa HTTPS bo'lishi kerak.

Avtomatik qayta-ulanish (tarmoq uzilsa) namunasi ham `CameraBroadcast.tsx` da bor —
`connectionstatechange` ni kuzatib exponential backoff bilan qayta ulanadi.

---

## 4. Flutter (mobil) — WHIP orqali

Flutter'da tayyor "WHIPClient" yo'q, lekin WHIP — oddiy protokol: **SDP offer'ni HTTP POST qilasiz, answer qaytadi**.
`flutter_webrtc` bilan to'liq ishlaydi.

```yaml
# pubspec.yaml
dependencies:
  flutter_webrtc: ^0.12.0
  http: ^1.2.0
```

```dart
import 'dart:async';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:http/http.dart' as http;

class WhipBroadcaster {
  RTCPeerConnection? _pc;
  MediaStream? _stream;
  String? _resourceUrl; // WHIP resurs URL (to'xtatish uchun)

  /// Kamera oqimini WHIP endpointga uzatadi.
  /// endpoint = "${whipBaseUrl}/$streamId/whip"
  Future<MediaStream> start(String endpoint) async {
    _stream = await navigator.mediaDevices.getUserMedia({
      'audio': true,
      'video': {
        'facingMode': 'user',
        'width': {'ideal': 1280},
        'height': {'ideal': 720},
        'frameRate': {'ideal': 30},
      },
    });

    _pc = await createPeerConnection({
      'iceServers': [
        {'urls': 'stun:stun.l.google.com:19302'},
        {'urls': 'stun:stun1.l.google.com:19302'},
      ],
    });

    // Kamera/mikrofon treklarini qo'shamiz (faqat yuborish).
    for (final track in _stream!.getTracks()) {
      await _pc!.addTrack(track, _stream!);
    }

    // SDP offer yaratamiz.
    final offer = await _pc!.createOffer();
    await _pc!.setLocalDescription(offer);

    // ICE to'planishini kutamiz (oddiy yo'l — to'liq to'planguncha).
    await _waitForIceGathering();
    final localSdp = (await _pc!.getLocalDescription())!.sdp!;

    // WHIP: SDP'ni application/sdp sifatida POST qilamiz.
    final res = await http.post(
      Uri.parse(endpoint),
      headers: {'Content-Type': 'application/sdp'},
      body: localSdp,
    );
    if (res.statusCode != 201 && res.statusCode != 200) {
      throw Exception('WHIP ulanmadi: ${res.statusCode} ${res.body}');
    }

    // 201 javobda answer SDP body'da, resurs URL Location header'da.
    _resourceUrl = res.headers['location'];
    await _pc!.setRemoteDescription(
      RTCSessionDescription(res.body, 'answer'),
    );

    return _stream!; // RTCVideoRenderer ga preview uchun ulang
  }

  Future<void> stop() async {
    // WHIP resursni o'chirish (server tomonda oqimni to'xtatadi).
    if (_resourceUrl != null) {
      final uri = _resourceUrl!.startsWith('http')
          ? Uri.parse(_resourceUrl!)
          : Uri.parse(_resourceUrl!); // kerak bo'lsa base bilan birlashtiring
      try {
        await http.delete(uri);
      } catch (_) {}
    }
    await _pc?.close();
    _stream?.getTracks().forEach((t) => t.stop());
    await _stream?.dispose();
    _pc = null;
    _stream = null;
    _resourceUrl = null;
  }

  Future<void> _waitForIceGathering() async {
    if (_pc!.iceGatheringState ==
        RTCIceGatheringState.RTCIceGatheringStateComplete) {
      return;
    }
    final completer = Completer<void>();
    Timer(const Duration(seconds: 3), () {
      if (!completer.isCompleted) completer.complete(); // timeout fallback
    });
    _pc!.onIceGatheringState = (state) {
      if (state == RTCIceGatheringState.RTCIceGatheringStateComplete &&
          !completer.isCompleted) {
        completer.complete();
      }
    };
    await completer.future;
  }
}
```

Preview ko'rsatish:

```dart
final _renderer = RTCVideoRenderer();
await _renderer.initialize();

final stream = await broadcaster.start("${session.whipBaseUrl}/$streamId/whip");
_renderer.srcObject = stream;
await startStream(streamId);  // market API

// Widget:
RTCVideoView(_renderer, mirror: true, objectFit: RTCVideoViewFit.RTCVideoViewObjectFitCover)
```

### Android/iOS ruxsatlari

```xml
<!-- android/app/src/main/AndroidManifest.xml -->
<uses-permission android:name="android.permission.CAMERA"/>
<uses-permission android:name="android.permission.RECORD_AUDIO"/>
<uses-permission android:name="android.permission.INTERNET"/>
```
```xml
<!-- ios/Runner/Info.plist -->
<key>NSCameraUsageDescription</key>
<string>Jonli efir uchun kamera kerak</string>
<key>NSMicrophoneUsageDescription</key>
<string>Jonli efir uchun mikrofon kerak</string>
```

---

## 5. RTMP varianti (OBS yoki ilova ichida)

WHIP yetarli bo'lmasa (masalan, professional overlaylar kerak) — RTMP.

**OBS sozlamasi:**
- Settings → Stream → Service: **Custom**
- Server: `rtmp://38.242.216.233:1935/live`  (`ingest.rtmp_url`)
- Stream Key: `sk_live_xxxxxxxxxxxxxxxxxxxxx`  (`ingest.stream_key`)

**Flutter'da RTMP** (kerak bo'lsa) — `apivideo_live_stream` paketi:
```yaml
dependencies:
  apivideo_live_stream: ^1.2.0
```
```dart
final controller = ApiVideoLiveStreamController(
  initialAudioConfig: AudioConfig(),
  initialVideoConfig: VideoConfig.withDefaultBitrate(),
);
await controller.initialize();
await controller.startStreaming(
  streamKey: session.ingest.streamKey,
  url: session.ingest.rtmpUrl, // "rtmp://.../live"
);
```

> RTMP'da `start`/`end` chaqiruvlari WHIP'dagidek market API orqali bo'ladi — faqat publish
> usuli RTMP. Streamni yaratishda `ingest_protocol: "rtmp"` yuboring.

---

## 6. Live-shopping: efirda mahsulot ko'rsatish

Efir jonli paytida seller "hozir shu mahsulot" deb belgilaydi — barcha tomoshabin ekranida
real-vaqtda karta chiqadi. Bu **Sahiy-Stream**'ga to'g'ridan-to'g'ri ketadi (`access_token` bilan).

**Mahsulotni ko'rsatish:**
```http
POST {whip_base_url}/v1/chat/{streamId}/featured
Authorization: Bearer {access_token}
Content-Type: application/json

{ "product_id": "123", "title": "Premium ko'ylak", "price": 150000,
  "image_url": "https://.../img.jpg", "url": "https://sahiy.uz/product/123" }
```

**Yashirish:**
```http
DELETE {whip_base_url}/v1/chat/{streamId}/featured
Authorization: Bearer {access_token}
```

**Joriy mahsulot (tomoshabin, auth shart emas):**
```http
GET {whip_base_url}/v1/chat/{streamId}/featured  →  { "data": { ... } | null }
```

Tomoshabin tomonida real-vaqt yangilanish chat WebSocket orqali keladi —
`{ "type": "featured_product", "product": {...} }` event'i. Batafsil:
`frontend/src/components/chat/ChatPanel.tsx` va `FeaturedProductCard.tsx`.

Flutter'da: chat WS (`wss://stream.vibrant.uz/v1/chat/{streamId}`) ga ulanib,
`type == "featured_product"` xabarini tinglang va video ustida karta ko'rsating.

---

## 7. Tez-tez uchraydigan xatolar

| Belgi | Sabab | Yechim |
|-------|-------|--------|
| `getUserMedia` ishlamaydi | HTTP (HTTPS emas) | Sahifa/ilova HTTPS bo'lsin yoki `localhost` |
| `start` "live" lekin video yo'q | `start` WHIP ulanishdan oldin chaqirilgan | Avval `ingest`/WHIP, keyin `start` |
| WHIP 404 | `streamId` noto'g'ri yoki stream yaratilmagan | Avval `POST .../streams/` qilib `id` oling |
| Tarmoq uzilib efir to'xtaydi | reconnect yo'q | `connectionstatechange` kuzatib qayta ulang (React namunasi) |
| 404 + HTML javob | market route'i stream domeniga yuborilgan | market chaqiruvlari market backend'iga, faqat WHIP/featured stream'ga |

---

## Bog'liq fayllar

- React WHIP: `frontend/src/components/broadcast/CameraBroadcast.tsx`
- WHIP endpoint helper: `frontend/src/lib/whip.ts`
- Featured (live-shopping): `frontend/src/lib/api/featured.ts`, `FeaturedProductCard.tsx`
- OBS barqaror efir: `docs/OBS-STABLE-STREAM.md`
- API ro'yxati: `docs/API.md`
