import { motion } from "motion/react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { useModalA11y } from "@/hooks/useModalA11y";

interface ConfirmDialogProps {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  isConfirming?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  title,
  message,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  danger,
  isConfirming,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  useModalA11y(onCancel);

  return (
    <motion.div
      className="fixed inset-0 z-20 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
      onClick={onCancel}
      role="presentation"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.15 }}
    >
      <Card
        role="alertdialog"
        aria-modal="true"
        aria-label={title}
        className="w-full max-w-sm p-6 shadow-lg"
        onClick={(e) => e.stopPropagation()}
        initial={{ opacity: 0, scale: 0.94, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.94, y: 8 }}
        transition={{ type: "spring", stiffness: 400, damping: 30 }}
      >
        <h2 className="text-base font-semibold text-fg">{title}</h2>
        <p className="mt-2 text-sm text-fg-muted">{message}</p>

        <div className="mt-5 flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onCancel} autoFocus>
            {cancelLabel}
          </Button>
          <Button
            type="button"
            variant={danger ? "danger" : "primary"}
            onClick={onConfirm}
            disabled={isConfirming}
          >
            {isConfirming ? "Working…" : confirmLabel}
          </Button>
        </div>
      </Card>
    </motion.div>
  );
}
