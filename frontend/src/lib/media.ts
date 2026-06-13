/** Brauzer kamera/mikrofon — faqat xavfsiz kontekstda (HTTPS yoki localhost). */
export function canUseCamera(): boolean {
  if (typeof window === "undefined") return false;
  return Boolean(navigator.mediaDevices?.getUserMedia) && window.isSecureContext;
}

export function cameraBlockedReason(): string | null {
  if (typeof window === "undefined") return null;
  if (!window.isSecureContext) {
    return "Kamera uchun HTTPS kerak. http:// IP manzilida telefon brauzeri kamerani bloklaydi. https://stream.shopla.uz manzilidan foydalaning yoki kompyuterda localhost ishlating.";
  }
  if (!navigator.mediaDevices?.getUserMedia) {
    return "Brauzeringiz kamera API ni qo'llab-quvvatlamaydi. Chrome yoki Safari yangi versiyasini ishlating.";
  }
  return null;
}
