import { cn } from "@/lib/utils";

export function Badge({
  children,
  variant = "default",
  className,
}: {
  children: React.ReactNode;
  variant?: "default" | "live" | "outline";
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-semibold uppercase tracking-wide",
        variant === "live" && "bg-live text-white shadow-lg shadow-live/30",
        variant === "outline" && "border border-border text-muted",
        variant === "default" && "bg-surface-3 text-foreground",
        className,
      )}
    >
      {variant === "live" && (
        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-white" />
      )}
      {children}
    </span>
  );
}
