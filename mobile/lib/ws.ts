import type { StreamEvent } from "./types";

const WS_BASE = process.env.EXPO_PUBLIC_WS_URL || "ws://localhost:8080";

export type StreamCallback = (event: StreamEvent) => void;

export function connectStream(
  streamID: string,
  onEvent: StreamCallback,
  onClose?: () => void,
): WebSocket {
  const ws = new WebSocket(
    `${WS_BASE}/api/v1/search/stream/${streamID}`,
  );

  ws.onmessage = (msg: MessageEvent) => {
    try {
      const event: StreamEvent = JSON.parse(
        typeof msg.data === "string" ? msg.data : "",
      );
      onEvent(event);
      if (event.type === "done" || event.type === "error") {
        ws.close();
      }
    } catch {
      // Ignore malformed messages
    }
  };

  ws.onerror = () => {
    ws.close();
  };

  ws.onclose = () => {
    onClose?.();
  };

  return ws;
}
