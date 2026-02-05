import type { RefObject } from "react";

export type TerminalStatus =
  | "idle"
  | "connecting"
  | "connected"
  | "error"
  | "closed";

export interface TerminalOptions {
  readonly wsUrl: string;
  readonly onStatusChange?: (status: TerminalStatus) => void;
  readonly onError?: (error: string) => void;
  readonly theme?: TerminalTheme;
  readonly fontSize?: number;
  readonly fontFamily?: string;
}

export interface TerminalTheme {
  readonly background: string;
  readonly foreground: string;
  readonly cursor: string;
  readonly selection?: string;
}

export interface UseTerminalReturn {
  readonly status: TerminalStatus;
  readonly error: string | null;
  readonly containerRef: RefObject<HTMLDivElement>;
  readonly connect: () => Promise<void>;
  readonly disconnect: () => void;
  readonly write: (data: string) => void;
  readonly sendInput: (data: string) => void;
  readonly clear: () => void;
}
