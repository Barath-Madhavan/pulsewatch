import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { connectStatusSocket } from "@/lib/websocket-client";
import type { CheckResult } from "@/types";

interface LiveStatusContextValue {
  statuses: Map<string, CheckResult>;
  connected: boolean;
}

const LiveStatusContext = createContext<LiveStatusContextValue | null>(null);

// Owns the single app-wide WebSocket connection and keeps the latest
// CheckResult per monitor, so every consumer sees live status without each
// opening its own socket.
export function LiveStatusProvider({ children }: { children: ReactNode }) {
  const [statuses, setStatuses] = useState<Map<string, CheckResult>>(new Map());
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const disconnect = connectStatusSocket({
      onStatusUpdate: (result) => {
        setStatuses((prev) => new Map(prev).set(result.monitor_id, result));
      },
      onConnectionChange: setConnected,
    });
    return disconnect;
  }, []);

  const value = useMemo(() => ({ statuses, connected }), [statuses, connected]);

  return (
    <LiveStatusContext.Provider value={value}>
      {children}
    </LiveStatusContext.Provider>
  );
}

function useLiveStatusContext() {
  const ctx = useContext(LiveStatusContext);
  if (!ctx) {
    throw new Error("useLiveStatus* hooks must be used within LiveStatusProvider");
  }
  return ctx;
}

export function useLiveStatuses(): Map<string, CheckResult> {
  return useLiveStatusContext().statuses;
}

export function useLiveStatus(monitorId: string): CheckResult | undefined {
  return useLiveStatusContext().statuses.get(monitorId);
}

export function useSocketConnected(): boolean {
  return useLiveStatusContext().connected;
}
