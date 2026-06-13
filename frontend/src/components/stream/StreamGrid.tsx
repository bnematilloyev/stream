import { StreamCard } from "./StreamCard";
import { Skeleton } from "@/components/ui/skeleton";
import type { Stream } from "@/types";

export function StreamGridSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="space-y-3">
          <Skeleton className="aspect-video w-full" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-3 w-1/2" />
        </div>
      ))}
    </div>
  );
}

export function StreamGrid({ streams }: { streams: Stream[] }) {
  if (streams.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border py-20 text-center">
        <p className="text-lg font-medium text-muted">Hozircha jonli stream yo&apos;q</p>
        <p className="mt-1 text-sm text-muted/70">
          Birinchi bo&apos;lib efirga chiqing!
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {streams.map((stream) => (
        <StreamCard key={stream.id} stream={stream} />
      ))}
    </div>
  );
}
