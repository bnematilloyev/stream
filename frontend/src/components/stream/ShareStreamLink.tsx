"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { watchPageUrl } from "@/lib/urls";
import { Check, Copy, Share2 } from "lucide-react";

interface ShareStreamLinkProps {
  streamId: string;
  title?: string;
}

export function ShareStreamLink({ streamId, title }: ShareStreamLinkProps) {
  const [copied, setCopied] = useState(false);
  const url = watchPageUrl(streamId);
  const canNativeShare =
    typeof navigator !== "undefined" && typeof navigator.share === "function";

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* fallback below */
    }
  }

  async function shareLink() {
    if (!canNativeShare) {
      await copyLink();
      return;
    }
    try {
      await navigator.share({
        url,
        title: title ?? "Jonli efir",
        text: title ? `${title} — jonli efirni tomosha qiling` : "Jonli efirni tomosha qiling",
      });
    } catch (e) {
      if (e instanceof Error && e.name === "AbortError") return;
      await copyLink();
    }
  }

  return (
    <div className="rounded-xl border border-border bg-surface-2 p-4 space-y-3">
      <p className="text-sm font-medium text-foreground">Tomosha linki</p>
      <p className="text-xs text-muted">
        Linkni do&apos;stlaringizga yuboring — ro&apos;yxatdan o&apos;tmasdan ham ko&apos;rishlari mumkin
      </p>
      <div className="flex gap-2">
        <Input readOnly value={url} className="font-mono text-xs" />
        <Button variant="secondary" size="icon" onClick={copyLink} aria-label="Nusxalash">
          {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
        </Button>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button variant="secondary" size="sm" onClick={copyLink} className="flex-1 sm:flex-none">
          <Copy className="h-4 w-4" />
          Nusxalash
        </Button>
        <Button size="sm" onClick={shareLink} className="flex-1 sm:flex-none">
          <Share2 className="h-4 w-4" />
          {canNativeShare ? "Ulashish" : "Linkni yuborish"}
        </Button>
      </div>
    </div>
  );
}
