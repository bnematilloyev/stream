"use client";

import { cn } from "@/lib/utils";

interface PlayerMessageProps {
  message: string;
  onRetry?: () => void;
  retryLabel?: string;
  className?: string;
}

/** Player ustidagi foydalanuvchiga tushunarli xabar. */
export function PlayerMessage({
  message,
  onRetry,
  retryLabel = "Qayta urinish",
  className,
}: PlayerMessageProps) {
  return (
    <div
      className={cn(
        "absolute inset-0 flex flex-col items-center justify-center gap-3 bg-black/80 px-6 text-center",
        className,
      )}
    >
      <p className="max-w-sm text-sm leading-relaxed text-white/90">{message}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white"
        >
          {retryLabel}
        </button>
      )}
    </div>
  );
}
