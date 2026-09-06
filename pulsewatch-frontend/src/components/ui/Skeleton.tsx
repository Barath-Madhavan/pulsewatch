import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

type SkeletonProps = HTMLAttributes<HTMLDivElement>;

export function Skeleton({ className, ...props }: SkeletonProps) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-surface-2", className)}
      aria-hidden="true"
      {...props}
    />
  );
}
