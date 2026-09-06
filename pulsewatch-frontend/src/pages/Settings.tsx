import { AlertCircle, Info, Mail, Monitor as MonitorIcon, Moon, Sun, Webhook } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { Switch } from "@/components/ui/Switch";
import { useAlertSettings, useUpdateAlertSettings } from "@/hooks/useAlertSettings";
import { useTheme, type ThemePreference } from "@/hooks/useTheme";
import { useToast } from "@/hooks/useToast";
import { cn } from "@/lib/utils";

const THEME_OPTIONS: { value: ThemePreference; label: string; icon: typeof Sun }[] = [
  { value: "light", label: "Light", icon: Sun },
  { value: "dark", label: "Dark", icon: Moon },
  { value: "system", label: "System", icon: MonitorIcon },
];

const inputClasses =
  "w-full rounded-lg border border-border bg-bg px-3 py-2 text-sm text-fg placeholder:text-fg-muted transition-colors focus-visible:border-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 disabled:opacity-50";
const labelClasses = "mb-1.5 block text-sm font-medium text-fg";

export function Settings() {
  const { data: settings, isLoading, isError, error } = useAlertSettings();
  const update = useUpdateAlertSettings();
  const { showToast } = useToast();
  const { preference, setPreference } = useTheme();

  const [emailEnabled, setEmailEnabled] = useState(false);
  const [emailAddress, setEmailAddress] = useState("");
  const [webhookEnabled, setWebhookEnabled] = useState(false);
  const [webhookUrl, setWebhookUrl] = useState("");

  // Hydrate the form from the server exactly once. Without the ref guard,
  // any background refetch (e.g. the window regaining focus) would re-run
  // this on every change to `settings` and silently overwrite whatever the
  // user was mid-typing with whatever the server has right now.
  const hydratedRef = useRef(false);
  useEffect(() => {
    if (!settings || hydratedRef.current) return;
    hydratedRef.current = true;
    setEmailEnabled(settings.email_enabled);
    setEmailAddress(settings.email_address);
    setWebhookEnabled(settings.webhook_enabled);
    setWebhookUrl(settings.webhook_url);
  }, [settings]);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    update.mutate(
      {
        email_enabled: emailEnabled,
        email_address: emailAddress,
        webhook_enabled: webhookEnabled,
        webhook_url: webhookUrl,
      },
      { onSuccess: () => showToast("Settings saved.") },
    );
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-7 w-28" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  if (isError) {
    return (
      <Card className="flex max-w-2xl animate-fade-in flex-col items-center gap-2 p-8 text-center">
        <AlertCircle className="size-6 text-down" />
        <p className="text-sm font-medium text-down">Couldn't load settings</p>
        <p className="mt-1 text-sm text-fg-muted">{(error as Error).message}</p>
      </Card>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="flex animate-fade-in flex-col gap-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-fg">Settings</h1>
        <p className="mt-1 text-sm text-fg-muted">
          Configure appearance and how alerts get delivered.
        </p>
      </div>

      <Card className="flex flex-wrap items-center justify-between gap-3 p-4">
        <p className="text-sm font-medium text-fg">Appearance</p>
        <div className="inline-flex w-fit rounded-lg border border-border bg-bg p-0.5">
          {THEME_OPTIONS.map(({ value, label, icon: Icon }) => (
            <button
              key={value}
              type="button"
              onClick={() => setPreference(value)}
              aria-pressed={preference === value}
              className={cn(
                "inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60",
                preference === value
                  ? "bg-accent/10 text-accent"
                  : "text-fg-muted hover:text-fg",
              )}
            >
              <Icon className="size-3.5" />
              {label}
            </button>
          ))}
        </div>
      </Card>

      {settings?.demo_mode && (
        <div className="flex items-start gap-3 rounded-xl border border-accent/30 bg-accent/10 p-3 text-sm text-accent">
          <Info className="mt-0.5 size-4 shrink-0" />
          <p>
            Email and webhook alerts are fully implemented. This is a public
            demo, so for security, sending is simulated instead of delivered
            to a real inbox or endpoint. Everything else, including the
            toggles, validation, and saving, works exactly as it would in
            production.
          </p>
        </div>
      )}

      <Card className="flex flex-col gap-4 p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3">
            <Mail className="mt-0.5 size-5 shrink-0 text-fg-muted" />
            <div>
              <p className="text-sm font-medium text-fg">Email alerts</p>
              <p className="text-sm text-fg-muted">
                Notify an email address when a monitor goes down or recovers.
              </p>
            </div>
          </div>
          <Switch
            checked={emailEnabled}
            onChange={setEmailEnabled}
            aria-label="Enable email alerts"
          />
        </div>

        <div>
          <label htmlFor="alert-email" className={labelClasses}>
            Email address
          </label>
          <input
            id="alert-email"
            type="email"
            className={inputClasses}
            placeholder="you@example.com"
            value={emailAddress}
            onChange={(e) => setEmailAddress(e.target.value)}
            disabled={!emailEnabled}
            required={emailEnabled}
          />
        </div>
      </Card>

      <Card className="flex flex-col gap-4 p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3">
            <Webhook className="mt-0.5 size-5 shrink-0 text-fg-muted" />
            <div>
              <p className="text-sm font-medium text-fg">Webhook alerts</p>
              <p className="text-sm text-fg-muted">
                POST a JSON payload to a URL when a monitor goes down or recovers.
              </p>
            </div>
          </div>
          <Switch
            checked={webhookEnabled}
            onChange={setWebhookEnabled}
            aria-label="Enable webhook alerts"
          />
        </div>

        <div>
          <label htmlFor="alert-webhook" className={labelClasses}>
            Webhook URL
          </label>
          <input
            id="alert-webhook"
            type="url"
            className={`${inputClasses} font-mono`}
            placeholder="https://example.com/hooks/pulsewatch"
            value={webhookUrl}
            onChange={(e) => setWebhookUrl(e.target.value)}
            disabled={!webhookEnabled}
            required={webhookEnabled}
          />
        </div>
      </Card>

      <div className="flex items-center gap-3">
        <Button type="submit" disabled={update.isPending}>
          {update.isPending ? "Saving…" : "Save"}
        </Button>
        {update.isError && (
          <p className="text-sm text-down">{(update.error as Error).message}</p>
        )}
      </div>
    </form>
  );
}
