import { ApiClientError } from "@/lib/api/client";

export type StreamStatus = "live" | "ended" | "scheduled" | string;

/** API va tarmoq xatolarini foydalanuvchiga ko'rsatish uchun. */
export function formatUserError(
  err: unknown,
  fallback = "Nimadir noto'g'ri ketdi. Qayta urinib ko'ring.",
): string {
  if (err instanceof ApiClientError) {
    switch (err.code) {
      case "NOT_FOUND":
        if (err.message.toLowerCase().includes("playback")) {
          return "Bu efir hozir ko'rilmaydi — tugagan yoki hali boshlanmagan bo'lishi mumkin.";
        }
        return "Ma'lumot topilmadi.";
      case "UNAUTHORIZED":
      case "FORBIDDEN":
        return "Bu amal uchun tizimga kirishingiz kerak.";
      case "VALIDATION_ERROR":
        return "Kiritilgan ma'lumot noto'g'ri. Tekshirib qayta urinib ko'ring.";
      default:
        break;
    }
    const msg = err.message.toLowerCase();
    if (msg.includes("playback not available")) {
      return "Bu efir hozir ko'rilmaydi — tugagan yoki hali boshlanmagan bo'lishi mumkin.";
    }
    if (msg.includes("network") || msg.includes("fetch")) {
      return "Internet bilan bog'lanishda muammo. Qayta urinib ko'ring.";
    }
    return fallback;
  }
  if (err instanceof Error) {
    const msg = err.message.toLowerCase();
    if (msg.includes("network") || msg.includes("fetch") || msg.includes("failed")) {
      return "Internet bilan bog'lanishda muammo. Qayta urinib ko'ring.";
    }
    if (msg.includes("whep") || msg.includes("hls") || msg.includes("webrtc")) {
      return "Videoni yuklab bo'lmadi. Qayta urinib ko'ring.";
    }
  }
  return fallback;
}

export function streamPreparingMessage(): string {
  return "Efir tayyorlanmoqda…";
}

export function streamNotLiveMessage(): string {
  return "Bu efir hozir jonli emas.";
}

export function streamEndedMessage(): string {
  return "Bu jonli efir tugagan.";
}

export function replayNotReadyMessage(): string {
  return "Yozuv hali tayyor emas yoki mavjud emas.";
}

export function whepPlaybackMessage(
  httpStatus: number | null,
  streamStatus?: StreamStatus,
): string {
  if (httpStatus === 404) {
    if (streamStatus === "ended") {
      return streamEndedMessage();
    }
    return "Translatsiya hozir mavjud emas. Efir tugagan yoki vaqtincha uzilgan bo'lishi mumkin.";
  }
  if (httpStatus === 403 || httpStatus === 401) {
    return "Bu efirni ko'rish uchun ruxsat yo'q.";
  }
  if (httpStatus != null && httpStatus >= 500) {
    return "Server vaqtincha javob bermayapti. Birozdan keyin qayta urinib ko'ring.";
  }
  if (streamStatus === "ended") {
    return streamEndedMessage();
  }
  return "Videoni yuklab bo'lmadi. Internetingizni tekshirib, qayta urinib ko'ring.";
}

export function hlsPlaybackMessage(opts: {
  manifest404?: boolean;
  streamEnded?: boolean;
  recovering?: boolean;
}): string {
  if (opts.streamEnded) {
    return streamEndedMessage();
  }
  if (opts.manifest404) {
    return "Efir hali tayyorlanmoqda. Bir necha soniya kuting yoki Ultra-low rejimini tanlang.";
  }
  if (opts.recovering) {
    return "Ulanish uzildi. Qayta ulanmoqda…";
  }
  return "Videoni ko'rsatib bo'lmadi. Qayta urinib ko'ring.";
}

export function connectionLostMessage(): string {
  return "Ulanish uzildi. Qayta urinib ko'ring.";
}

export function chatLoginRequiredMessage(): string {
  return "Chatda yozish uchun avval tizimga kiring.";
}

export function chatHistoryFailedMessage(): string {
  return "Chat xabarlari yuklanmadi. Sahifani yangilab ko'ring.";
}

export function chatUnavailableMessage(): string {
  return "Chat vaqtincha ishlamayapti. Video tomoshasi davom etadi.";
}

export function chatServerMessage(raw: string): string {
  const msg = raw.toLowerCase();
  if (msg.includes("authentication required") || msg.includes("auth")) {
    return chatLoginRequiredMessage();
  }
  if (msg.includes("rate") || msg.includes("slow")) {
    return "Juda tez xabar yuboryapsiz. Biroz kuting.";
  }
  if (msg.includes("invalid message")) {
    return "Xabar yuborib bo'lmadi. Qayta urinib ko'ring.";
  }
  return "Xabar yuborib bo'lmadi. Qayta urinib ko'ring.";
}

export function whipBroadcastMessage(err: unknown): string {
  if (err instanceof Error) {
    const msg = err.message.toLowerCase();
    if (msg.includes("fetch") || msg.includes("failed") || msg.includes("network")) {
      return "Kamera serveriga ulanib bo'lmadi. Internetni tekshirib, qayta urinib ko'ring.";
    }
    if (msg.includes("ruxsat") || msg.includes("permission") || msg.includes("denied")) {
      return "Kameraga ruxsat berilmadi. Brauzer sozlamalaridan kamera va mikrofonni yoqing.";
    }
    if (msg.includes("https") || msg.includes("qo'llab-quvvatlamaydi")) {
      return err.message;
    }
  }
  return formatUserError(err, "Efirni boshlab bo'lmadi. Qayta urinib ko'ring.");
}

export function browserNoHlsMessage(): string {
  return "Brauzeringiz bu formatdagi videoni qo'llab-quvvatlamaydi. Boshqa brauzerdan urinib ko'ring.";
}
