import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AnimatePresence, motion } from "motion/react";
import { useState } from "react";
import { BrowserRouter, Navigate, Route, Routes, useLocation } from "react-router-dom";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Sidebar } from "@/components/layout/Sidebar";
import { TopBar } from "@/components/layout/TopBar";
import { LiveStatusProvider } from "@/hooks/useWebSocket";
import { ThemeProvider } from "@/hooks/useTheme";
import { ToastProvider } from "@/hooks/useToast";
import { Dashboard } from "@/pages/Dashboard";
import { Incidents } from "@/pages/Incidents";
import { MonitorDetail } from "@/pages/MonitorDetail";
import { Settings } from "@/pages/Settings";

// AnimatePresence needs the outgoing page to keep rendering its old content
// while it fades out, so `location` is captured here and handed explicitly
// to `<Routes>` rather than left to resolve reactively against whatever the
// URL has already become.
function AnimatedRoutes() {
  const location = useLocation();
  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={location.pathname}
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -8 }}
        transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
      >
        <Routes location={location}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/monitors/:id" element={<MonitorDetail />} />
          <Route path="/incidents" element={<Incidents />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </motion.div>
    </AnimatePresence>
  );
}

function App() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // This is a local dashboard talking to a backend on the same
            // machine, where "unreachable" almost always means "not running
            // yet," not transient network flakiness. React Query's default
            // 3-retry backoff can take 10+ seconds before surfacing that,
            // which reads as a stuck loading state. One quick retry gets
            // an error on screen fast instead.
            retry: 1,
            retryDelay: 1000,
          },
        },
      }),
  );

  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <LiveStatusProvider>
            <BrowserRouter>
              <div className="flex h-screen bg-bg text-fg">
                <Sidebar />
                <div className="relative flex min-w-0 flex-1 flex-col">
                  {/* Ambient background glow, fixed to the viewport so any
                      leftover space (a wide monitor, a short page) reads as
                      deliberate atmosphere rather than empty/broken. Purely
                      decorative: behind everything, never intercepts clicks. */}
                  <div
                    className="pointer-events-none fixed inset-0 -z-10 overflow-hidden"
                    aria-hidden="true"
                  >
                    {/* A faint dot grid gives the whole app a subtle sense of
                        texture/depth (the same trick Linear and similar apps
                        use) instead of a completely flat, empty background.
                        var(--color-fg-muted) keeps it theme-adaptive; the
                        opacity is deliberately tiny so it reads as texture,
                        not as visible dots. */}
                    <div
                      className="absolute inset-0 opacity-[0.05]"
                      style={{
                        backgroundImage:
                          "radial-gradient(circle, var(--color-fg-muted) 1px, transparent 1px)",
                        backgroundSize: "24px 24px",
                      }}
                    />
                    <div className="absolute -top-40 right-0 size-[36rem] rounded-full bg-accent/[0.07] blur-3xl" />
                    <div className="absolute -bottom-48 left-1/4 size-[28rem] rounded-full bg-accent-2/[0.05] blur-3xl" />
                  </div>

                  <TopBar />
                  <main className="flex-1 overflow-y-auto overflow-x-hidden p-4 sm:p-6">
                    <div className="mx-auto w-full max-w-[1400px]">
                      <ErrorBoundary>
                        <AnimatedRoutes />
                      </ErrorBoundary>
                    </div>
                  </main>
                </div>
              </div>
            </BrowserRouter>
          </LiveStatusProvider>
        </ToastProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}

export default App;
