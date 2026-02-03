import type { SSEEvent } from "@/types";

type SSECallback = (event: SSEEvent) => void;

const API_URL = import.meta.env.VITE_API_URL ?? "";

class SSEClient {
  private eventSource: EventSource | null = null;
  private readonly callbacks: Set<SSECallback> = new Set();
  private reconnectAttempts = 0;
  private reconnectTimeout: NodeJS.Timeout | null = null;

  connect(url: string = `${API_URL}/events/deploys`): void {
    if (this.eventSource?.readyState === EventSource.OPEN) {
      return;
    }

    this.eventSource = new EventSource(url);

    this.eventSource.addEventListener("deploy", (event) => {
      this.handleEvent(event);
    });

    this.eventSource.addEventListener("log", (event) => {
      this.handleEvent(event);
    });

    this.eventSource.addEventListener("health", (event) => {
      this.handleEvent(event);
    });

    this.eventSource.addEventListener("stats", (event) => {
      this.handleEvent(event);
    });

    this.eventSource.addEventListener("provision", (event) => {
      this.handleEvent(event);
    });

    this.eventSource.onopen = () => {
      this.reconnectAttempts = 0;
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
    this.disconnect();

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;

    this.reconnectTimeout = setTimeout(() => {
      this.connect();
    }, delay);
  }

  subscribe(callback: SSECallback): () => void {
    this.callbacks.add(callback);
    return () => this.callbacks.delete(callback);
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
  }

  get isConnected(): boolean {
    return this.eventSource?.readyState === EventSource.OPEN;
  }
}

export const sseClient = new SSEClient();
