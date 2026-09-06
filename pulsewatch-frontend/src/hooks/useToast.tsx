import { CheckCircle2, XCircle } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useState,
} from "react";

type ToastVariant = "success" | "error";

interface ToastItem {
  id: number;
  message: string;
  variant: ToastVariant;
}

interface ToastContextValue {
  showToast: (message: string, variant?: ToastVariant) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

const TOAST_DURATION_MS = 3000;
let nextToastId = 0;

// A single app-wide toast stack, shown bottom-right. Used for transient
// confirmations (e.g. "Settings saved.") that don't need to stay on
// screen the way an inline validation error does.
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const showToast = useCallback((message: string, variant: ToastVariant = "success") => {
    const id = nextToastId++;
    setToasts((prev) => [...prev, { id, message, variant }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, TOAST_DURATION_MS);
  }, []);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      <div className="pointer-events-none fixed bottom-6 right-6 z-50 flex flex-col gap-2">
        <AnimatePresence>
          {toasts.map((t) => (
            <motion.div
              key={t.id}
              role="status"
              layout
              initial={{ opacity: 0, x: 32, scale: 0.9 }}
              animate={{ opacity: 1, x: 0, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9, transition: { duration: 0.15 } }}
              transition={{ type: "spring", stiffness: 400, damping: 30 }}
              className="pointer-events-auto flex items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-fg shadow-lg"
            >
              {t.variant === "success" ? (
                <CheckCircle2 className="size-4 shrink-0 text-up" />
              ) : (
                <XCircle className="size-4 shrink-0 text-down" />
              )}
              {t.message}
            </motion.div>
          ))}
        </AnimatePresence>
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return ctx;
}
