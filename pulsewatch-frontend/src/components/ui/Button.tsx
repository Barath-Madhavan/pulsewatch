import { forwardRef, type ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    "bg-gradient-to-b from-accent to-[color-mix(in_srgb,var(--color-accent)_85%,var(--color-accent-2))] text-white shadow-sm hover:shadow-md hover:brightness-105",
  secondary: "border border-border bg-surface text-fg shadow-sm hover:bg-surface-2",
  ghost: "text-fg-muted hover:bg-surface-2 hover:text-fg",
  danger: "border border-down/20 bg-down/10 text-down hover:bg-down/20",
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = "primary", className, ...props },
  ref,
) {
  return (
    <button
      ref={ref}
      className={cn(
        "inline-flex shrink-0 items-center justify-center gap-2 rounded-lg px-3.5 py-2 text-sm font-medium whitespace-nowrap transition-[background,box-shadow,transform,filter] duration-150 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 focus-visible:ring-offset-2 focus-visible:ring-offset-bg",
        variantClasses[variant],
        className,
      )}
      {...props}
    />
  );
});
