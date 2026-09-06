import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  pageSizeOptions: readonly number[];
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
}

export function Pagination({
  page,
  pageSize,
  total,
  pageSizeOptions,
  onPageChange,
  onPageSizeChange,
}: PaginationProps) {
  const start = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);
  const canGoPrev = page > 1;
  const canGoNext = end < total;

  return (
    <div className="flex flex-wrap items-center justify-end gap-4 text-sm text-fg-muted">
      <label className="flex items-center gap-2">
        Rows per page
        <select
          value={pageSize}
          onChange={(e) => onPageSizeChange(Number(e.target.value))}
          className="rounded-lg border border-border bg-bg px-2 py-1 text-sm text-fg transition-colors focus-visible:border-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
        >
          {pageSizeOptions.map((size) => (
            <option key={size} value={size}>
              {size}
            </option>
          ))}
        </select>
      </label>

      <span className="tabular-nums">
        {start}-{end} of {total}
      </span>

      <div className="flex items-center gap-1">
        <button
          type="button"
          aria-label="Previous page"
          onClick={() => onPageChange(page - 1)}
          disabled={!canGoPrev}
          className={cn(
            "rounded-md p-1.5 transition-colors hover:bg-surface-2 hover:text-fg",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60",
            "disabled:pointer-events-none disabled:opacity-40",
          )}
        >
          <ChevronLeft className="size-4" />
        </button>
        <button
          type="button"
          aria-label="Next page"
          onClick={() => onPageChange(page + 1)}
          disabled={!canGoNext}
          className={cn(
            "rounded-md p-1.5 transition-colors hover:bg-surface-2 hover:text-fg",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60",
            "disabled:pointer-events-none disabled:opacity-40",
          )}
        >
          <ChevronRight className="size-4" />
        </button>
      </div>
    </div>
  );
}
