import { Skeleton } from "@/components/ui/skeleton";

export function MetricValueSkeleton({ className }: { className?: string }) {
  return <Skeleton aria-hidden className={className ?? "h-8 w-24"} />;
}

export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  const keys = Array.from({ length: Math.max(0, rows) }, (_, i) => `row-${i}`);
  return (
    <div className="space-y-2">
      {keys.map((key) => (
        <Skeleton key={key} className="h-10 w-full" />
      ))}
    </div>
  );
}
