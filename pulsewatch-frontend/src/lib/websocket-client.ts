import type { CheckResult, StatusUpdateMessage } from "@/types";

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";
const WS_URL = `${API_URL.replace(/^http/, "ws")}/ws`;

const RECONNECT_DELAY_MS = 3000;

interface ConnectOptions {
  onStatusUpdate: (result: CheckResult) => void;
  onConnectionChange?: (connected: boolean) => void;
}

// Opens a live connection to the backend's /ws endpoint and reconnects on
// drop. Returns a cleanup function that closes it for good.
export function connectStatusSocket({
  onStatusUpdate,
  onConnectionChange,
}: ConnectOptions): () => void {
  let socket: WebSocket | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
  let closedByCaller = false;

  function connect() {
    socket = new WebSocket(WS_URL);

    socket.onopen = () => onConnectionChange?.(true);

    socket.onmessage = (event) => {
      let message: StatusUpdateMessage;
      try {
        message = JSON.parse(event.data);
      } catch {
        return;
      }
      if (message.type === "status_update") {
        onStatusUpdate(message.payload);
      }
    };

    socket.onclose = () => {
      onConnectionChange?.(false);
      if (!closedByCaller) {
        reconnectTimer = setTimeout(connect, RECONNECT_DELAY_MS);
      }
    };

    socket.onerror = () => {
      socket?.close();
    };
  }

  connect();

  return () => {
    closedByCaller = true;
    clearTimeout(reconnectTimer);
    socket?.close();
  };
}
