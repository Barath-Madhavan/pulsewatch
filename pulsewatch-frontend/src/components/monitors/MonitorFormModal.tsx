import { X } from "lucide-react";
import { motion } from "motion/react";
import { type FormEvent, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { useCreateMonitor, useUpdateMonitor } from "@/hooks/useMonitors";
import { useModalA11y } from "@/hooks/useModalA11y";
import type { Monitor } from "@/types";

interface MonitorFormModalProps {
  monitor?: Monitor;
  onClose: () => void;
}

const inputClasses =
  "w-full rounded-lg border border-border bg-bg px-3 py-2 text-sm text-fg placeholder:text-fg-muted transition-colors focus-visible:border-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60";
const labelClasses = "mb-1.5 block text-sm font-medium text-fg whitespace-nowrap";

export function MonitorFormModal({ monitor, onClose }: MonitorFormModalProps) {
  const isEditing = Boolean(monitor);
  const createMonitor = useCreateMonitor();
  const updateMonitor = useUpdateMonitor(monitor?.id ?? "");
  const mutation = isEditing ? updateMonitor : createMonitor;

  useModalA11y(onClose);

  const [name, setName] = useState(monitor?.name ?? "");
  const [url, setUrl] = useState(monitor?.url ?? "");
  const [method, setMethod] = useState(monitor?.method ?? "GET");
  const [intervalSeconds, setIntervalSeconds] = useState(monitor?.interval_seconds ?? 60);
  const [timeoutSeconds, setTimeoutSeconds] = useState(monitor?.timeout_seconds ?? 10);
  const [expectedStatusCode, setExpectedStatusCode] = useState(
    monitor?.expected_status_code ?? 200,
  );

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const payload = {
      name,
      url,
      method,
      interval_seconds: intervalSeconds,
      timeout_seconds: timeoutSeconds,
      expected_status_code: expectedStatusCode,
    };
    mutation.mutate(payload, { onSuccess: onClose });
  }

  return (
    <motion.div
      className="fixed inset-0 z-20 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
      onClick={onClose}
      role="presentation"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.15 }}
    >
      <Card
        role="dialog"
        aria-modal="true"
        aria-label={isEditing ? "Edit monitor" : "Add monitor"}
        className="w-full max-w-md p-6 shadow-lg"
        onClick={(e) => e.stopPropagation()}
        initial={{ opacity: 0, scale: 0.94, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.94, y: 8 }}
        transition={{ type: "spring", stiffness: 400, damping: 30 }}
      >
        <div className="mb-5 flex items-center justify-between">
          <h2 className="text-base font-semibold text-fg">
            {isEditing ? "Edit Monitor" : "Add Monitor"}
          </h2>
          <button
            type="button"
            aria-label="Close"
            onClick={onClose}
            className="rounded-md p-1 text-fg-muted transition-colors hover:bg-surface-2 hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
          >
            <X className="size-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div>
            <label htmlFor="monitor-name" className={labelClasses}>
              Name
            </label>
            <input
              id="monitor-name"
              className={inputClasses}
              placeholder="Production API"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={200}
              required
              autoFocus
            />
          </div>

          <div>
            <label htmlFor="monitor-url" className={labelClasses}>
              URL
            </label>
            <input
              id="monitor-url"
              type="url"
              className={`${inputClasses} font-mono`}
              placeholder="https://example.com/health"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              required
            />
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div>
              <label htmlFor="monitor-method" className={labelClasses}>
                Method
              </label>
              <select
                id="monitor-method"
                className={inputClasses}
                value={method}
                onChange={(e) => setMethod(e.target.value)}
              >
                <option>GET</option>
                <option>POST</option>
                <option>HEAD</option>
              </select>
            </div>

            <div>
              <label htmlFor="monitor-interval" className={labelClasses}>
                Interval (s)
              </label>
              <input
                id="monitor-interval"
                type="number"
                min={5}
                max={86400}
                className={inputClasses}
                value={intervalSeconds}
                onChange={(e) => setIntervalSeconds(Number(e.target.value))}
                required
              />
            </div>

            <div>
              <label htmlFor="monitor-status-code" className={labelClasses}>
                Status code
              </label>
              <input
                id="monitor-status-code"
                type="number"
                min={100}
                max={599}
                className={inputClasses}
                value={expectedStatusCode}
                onChange={(e) => setExpectedStatusCode(Number(e.target.value))}
                required
              />
            </div>
          </div>

          <div>
            <label htmlFor="monitor-timeout" className={labelClasses}>
              Timeout (s)
            </label>
            <input
              id="monitor-timeout"
              type="number"
              min={1}
              max={300}
              className={inputClasses}
              value={timeoutSeconds}
              onChange={(e) => setTimeoutSeconds(Number(e.target.value))}
              required
            />
            {timeoutSeconds > intervalSeconds && (
              <p className="mt-1.5 text-xs text-degraded">
                Timeout can't be longer than the check interval.
              </p>
            )}
          </div>

          {mutation.isError && (
            <p className="text-sm text-down">{(mutation.error as Error).message}</p>
          )}

          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={mutation.isPending || timeoutSeconds > intervalSeconds}
            >
              {mutation.isPending
                ? isEditing
                  ? "Saving…"
                  : "Adding…"
                : isEditing
                  ? "Save Changes"
                  : "Add Monitor"}
            </Button>
          </div>
        </form>
      </Card>
    </motion.div>
  );
}
