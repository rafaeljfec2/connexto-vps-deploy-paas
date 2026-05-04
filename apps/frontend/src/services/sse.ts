import type { SSEEvent } from "@/types";

type SSECallback = (event: SSEEvent) => void;
type ConnectionListener = (connected: boolean) => void;
type EventSourceFactory = (url: string) => EventSource;

const API_URL = import.meta.env.VITE_API_URL ?? "";

const SSE_EVENT_NAMES = [
  "deploy",
  "log",
  "health",
  "stats",
  "system_stats",
  "server_stats",
  "invalidate",
  "provision",
  "agent_update",
] as const;

function defaultEventSourceFactory(url: string): EventSource {
  return new EventSource(url, { withCredentials: true });
}

class SSEClient {
  private eventSource: EventSource | null = null;
  private readonly callbacks: Set<SSECallback> = new Set();
  private readonly connectionListeners: Set<ConnectionListener> = new Set();
  private readonly createEventSource: EventSourceFactory;
  private reconnectAttempts = 0;
  private reconnectTimeout: NodeJS.Timeout | null = null;
  private connectedSnapshot = false;

  constructor(
    createEventSource: EventSourceFactory = defaultEventSourceFactory,
  ) {
    this.createEventSource = createEventSource;
  }

  connect(url: string = `${API_URL}/events/deploys`): void {
    if (this.eventSource?.readyState === EventSource.OPEN) {
      return;
    }

    this.eventSource = this.createEventSource(url);

    for (const name of SSE_EVENT_NAMES) {
      this.eventSource.addEventListener(name, (event) => {
        this.handleEvent(event);
      });
    }

    this.eventSource.onopen = () => {
      this.reconnectAttempts = 0;
      this.setConnected(true);
    };

    this.eventSource.onerror = () => {
      this.handleError();
    };
  }

  private handleEvent(event: MessageEvent): void {
    try {
      const data: SSEEvent = JSON.parse(event.data);
      this.callbacks.forEach((callback) => callback(data));
    } catch (error) {
      console.error("Failed to parse SSE event:", error);
    }
  }

  private handleError(): void {
    const wasConnected = this.eventSource?.readyState === EventSource.OPEN;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);

    console.warn("[sse] connection error, scheduling reconnect", {
      wasConnected,
      attempts: this.reconnectAttempts,
      retryInMs: delay,
    });

    this.disconnect();
    this.reconnectAttempts++;

    this.reconnectTimeout = setTimeout(() => {
      this.connect();
    }, delay);
  }

  subscribe(callback: SSECallback): () => void {
    this.callbacks.add(callback);
    return () => this.callbacks.delete(callback);
  }

  /**
   * Subscribe to SSE connection-state changes (open/closed). Listener is
   * invoked synchronously with the current state on subscribe and then on
   * every transition. Returns an unsubscribe function.
   */
  subscribeConnectionState(listener: ConnectionListener): () => void {
    this.connectionListeners.add(listener);
    listener(this.connectedSnapshot);
    return () => this.connectionListeners.delete(listener);
  }

  disconnect(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    this.setConnected(false);
  }

  private setConnected(connected: boolean): void {
    if (this.connectedSnapshot === connected) return;
    this.connectedSnapshot = connected;
    this.connectionListeners.forEach((listener) => listener(connected));
  }

  get isConnected(): boolean {
    return this.connectedSnapshot;
  }
}

export { SSEClient };
export type { EventSourceFactory };
export const sseClient = new SSEClient();
