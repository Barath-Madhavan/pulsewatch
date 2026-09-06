import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type BadgeVariant = "up" | "down" | "degraded" | "neutral" | "accent";

const variantClasses: Record<BadgeVariant, string> = {
  up: "bg-up/10 text-up border-up/20",
  down: "bg-down/10 text-down border-down/20",
  degraded: "bg-degraded/10 text-degraded border-degraded/20",
  neutral: "bg-border/40 text-fg-muted border-border",
  accent: "bg-accent/10 text-accent border-accent/20",
};

interface BadgeProps {
  variant?: BadgeVariant;
  children: ReactNode;
  className?: string;
}

export function Badge({ variant = "neutral", children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium whitespace-nowrap transition-colors",
        variantClasses[variant],
        className,
      )}
    >
      {children}
    </span>
  );
}
