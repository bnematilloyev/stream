import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { formatViewerCount } from "@/lib/utils";
import type { Stream } from "@/types";

export function StreamCard({ stream }: { stream: Stream }) {
  const isLive = stream.status === "live";

  return (
    <Link
      href={`/live/${stream.id}`}
      className="group block overflow-hidden rounded-2xl border border-border/50 bg-surface-1 transition-all duration-300 hover:-translate-y-1 hover:border-brand-secondary/30 hover:shadow-xl hover:shadow-brand/5"
    >
      <div className="relative aspect-video overflow-hidden bg-surface-2">
        <div className="absolute inset-0 bg-gradient-to-br from-brand/10 via-surface-2 to-brand-secondary/10 transition-transform duration-500 group-hover:scale-105" />
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-black/40 backdrop-blur-sm transition-transform group-hover:scale-110">
            <div className="ml-1 h-0 w-0 border-y-[10px] border-l-[16px] border-y-transparent border-l-white" />
          </div>
        </div>
        {isLive && (
          <div className="absolute left-3 top-3">
            <Badge variant="live">Live</Badge>
          </div>
        )}
        {stream.viewer_count > 0 && (
          <div className="absolute bottom-3 right-3 rounded-lg bg-black/70 px-2 py-1 text-xs font-medium text-white backdrop-blur-sm">
            {formatViewerCount(stream.viewer_count)} tomoshabin
          </div>
        )}
      </div>
      <div className="p-4">
        <h3 className="line-clamp-2 font-semibold leading-snug text-foreground transition-colors group-hover:text-brand-secondary">
          {stream.title}
        </h3>
        <p className="mt-1 text-sm text-muted">{stream.channel_title}</p>
      </div>
    </Link>
  );
}
