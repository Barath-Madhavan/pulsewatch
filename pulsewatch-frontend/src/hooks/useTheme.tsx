import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

export type ThemePreference = "light" | "dark" | "system";
type ResolvedTheme = "light" | "dark";

const STORAGE_KEY = "pulsewatch-theme";
const DARK_QUERY = "(prefers-color-scheme: dark)";

interface ThemeContextValue {
  preference: ThemePreference;
  resolvedTheme: ResolvedTheme;
  setPreference: (preference: ThemePreference) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

function readStoredPreference(): ThemePreference {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "light" || stored === "dark" || stored === "system") return stored;
  } catch {
    /* localStorage unavailable (private mode, etc.); fall through to system */
  }
  return "system";
}

function resolve(preference: ThemePreference): ResolvedTheme {
  if (preference === "light" || preference === "dark") return preference;
  return window.matchMedia(DARK_QUERY).matches ? "dark" : "light";
}

function applyTheme(resolved: ResolvedTheme) {
  document.documentElement.dataset.theme = resolved;
  document.documentElement.style.colorScheme = resolved;
}

// Context+hook, same shape as ToastProvider/LiveStatusProvider. Persists the
// user's Light/Dark/System choice, resolves "system" against the OS setting
// live, and cross-fades the swap via the View Transitions API when the
// browser supports it and the user hasn't asked for reduced motion.
export function ThemeProvider({ children }: { children: ReactNode }) {
  const [preference, setPreferenceState] = useState<ThemePreference>(readStoredPreference);
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>(() =>
    resolve(readStoredPreference()),
  );

  useEffect(() => {
    if (preference !== "system") return;
    const mql = window.matchMedia(DARK_QUERY);
    function handleChange() {
      const next = resolve("system");
      setResolvedTheme(next);
      applyTheme(next);
    }
    mql.addEventListener("change", handleChange);
    return () => mql.removeEventListener("change", handleChange);
  }, [preference]);

  const setPreference = useCallback((next: ThemePreference) => {
    const nextResolved = resolve(next);

    const commit = () => {
      try {
        localStorage.setItem(STORAGE_KEY, next);
      } catch {
        /* non-fatal; theme just won't persist across reloads */
      }
      setPreferenceState(next);
      setResolvedTheme(nextResolved);
      applyTheme(nextResolved);
    };

    const prefersReducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    const supportsViewTransition =
      typeof document.startViewTransition === "function";

    if (supportsViewTransition && !prefersReducedMotion) {
      document.startViewTransition(commit);
    } else {
      commit();
    }
  }, []);

  return (
    <ThemeContext.Provider value={{ preference, resolvedTheme, setPreference }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return ctx;
}

const CHART_COLOR_VARS = {
  accent: "--color-accent",
  border: "--color-border",
  surface: "--color-surface",
  fgMuted: "--color-fg-muted",
  up: "--color-up",
  down: "--color-down",
  degraded: "--color-degraded",
} as const;

export interface ChartColors {
  accent: string;
  border: string;
  surface: string;
  fgMuted: string;
  up: string;
  down: string;
  degraded: string;
}

// Recharts takes literal color strings, not Tailwind classes, so it can't
// pick up a CSS variable swap on its own. Re-reads the resolved custom
// properties whenever the theme changes so charts repaint correctly in both.
export function useThemeColors(): ChartColors {
  const { resolvedTheme } = useTheme();
  const [colors, setColors] = useState<ChartColors>(() => readChartColors());

  useEffect(() => {
    setColors(readChartColors());
  }, [resolvedTheme]);

  return colors;
}

function readChartColors(): ChartColors {
  const styles = getComputedStyle(document.documentElement);
  const read = (name: string) => styles.getPropertyValue(name).trim();
  return {
    accent: read(CHART_COLOR_VARS.accent),
    border: read(CHART_COLOR_VARS.border),
    surface: read(CHART_COLOR_VARS.surface),
    fgMuted: read(CHART_COLOR_VARS.fgMuted),
    up: read(CHART_COLOR_VARS.up),
    down: read(CHART_COLOR_VARS.down),
    degraded: read(CHART_COLOR_VARS.degraded),
  };
}
