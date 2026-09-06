import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Card } from "@/components/ui/Card";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: ReactNode;
}

export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <Card className="flex animate-fade-in flex-col items-center gap-3 p-12 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-surface-2">
        <Icon className="size-6 text-fg-muted" />
      </div>
      <p className="text-sm font-medium text-fg">{title}</p>
      <p className="max-w-sm text-sm text-fg-muted">{description}</p>
      {action}
    </Card>
  );
}
