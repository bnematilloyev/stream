import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { formatViewerCount } from "@/lib/utils";
import type { Channel } from "@/types";
import { FollowButton } from "./FollowButton";

export function ChannelHeader({ channel }: { channel: Channel }) {
  return (
    <div className="relative overflow-hidden rounded-2xl border border-border bg-surface-1">
      <div className="brand-bg-alt h-32 sm:h-40" />
      <div className="relative px-6 pb-6">
        <div className="-mt-10 flex flex-col gap-4 sm:-mt-12 sm:flex-row sm:items-end sm:justify-between">
          <div className="flex items-end gap-4">
            <div className="accent-gradient flex h-20 w-20 shrink-0 items-center justify-center rounded-2xl border-4 border-surface-1 text-2xl font-bold text-brand shadow-xl sm:h-24 sm:w-24">
              {channel.title.charAt(0).toUpperCase()}
            </div>
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="text-2xl font-bold">{channel.title}</h1>
                {channel.is_verified && (
                  <Badge variant="outline">Tasdiqlangan</Badge>
                )}
                {channel.is_live && <Badge variant="live">Live</Badge>}
              </div>
              <p className="text-muted">@{channel.slug}</p>
              <p className="mt-1 text-sm text-muted">
                {formatViewerCount(channel.follower_count)} obunachi
              </p>
            </div>
          </div>
          <div className="flex gap-2">
            <FollowButton slug={channel.slug} />
            <Link
              href={`/channel/${channel.slug}`}
              className="rounded-xl border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-surface-2"
            >
              Streamlar
            </Link>
          </div>
        </div>
        {channel.description && (
          <p className="mt-4 max-w-2xl text-sm text-muted leading-relaxed">
            {channel.description}
          </p>
        )}
      </div>
    </div>
  );
}
