import { motion } from "motion/react";
import type { HTMLMotionProps } from "motion/react";
import { cn } from "@/lib/utils";

interface CardProps extends HTMLMotionProps<"div"> {
  /** Adds a subtle spring-driven hover lift, for cards that are themselves a click target. */
  interactive?: boolean;
}

export function Card({ className, interactive, ...props }: CardProps) {
  return (
    <motion.div
      className={cn("rounded-xl border border-border bg-surface shadow-sm", className)}
      whileHover={interactive ? { y: -4, boxShadow: "var(--shadow-md)" } : undefined}
      transition={{ type: "spring", stiffness: 400, damping: 28 }}
      {...props}
    />
  );
}
