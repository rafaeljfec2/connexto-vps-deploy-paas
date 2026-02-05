import type { TerminalTheme } from "./types";

export function keyEventToData(event: KeyboardEvent): string {
  if (event.ctrlKey && event.key.length === 1) {
    const code = (event.key.toUpperCase().codePointAt(0) ?? 0) - 64;
    if (code >= 1 && code <= 26) return String.fromCodePoint(code);
  }
  if (event.altKey || event.metaKey) return "";

  switch (event.key) {
    case "Enter":
      return "\r";
    case "Backspace":
      return "\x7f";
    case "Tab":
      return "\t";
    case "Escape":
      return "\x1b";
    case "ArrowUp":
      return "\x1b[A";
    case "ArrowDown":
      return "\x1b[B";
    case "ArrowRight":
      return "\x1b[C";
    case "ArrowLeft":
      return "\x1b[D";
    case "Home":
      return "\x1b[H";
    case "End":
      return "\x1b[F";
    case "Delete":
      return "\x1b[3~";
    case "PageUp":
      return "\x1b[5~";
    case "PageDown":
      return "\x1b[6~";
    default:
      return event.key.length === 1 ? event.key : "";
  }
}

export function isBrowserShortcut(event: KeyboardEvent): boolean {
  const key = event.key.toLowerCase();
  const shortcuts = ["r", "w", "t", "n", "l", "d", "f"];
  return (event.ctrlKey || event.metaKey) && shortcuts.includes(key);
}

export const DEFAULT_THEME: TerminalTheme = {
  background: "#0f172a",
  foreground: "#e2e8f0",
  cursor: "#94a3b8",
  selection: "rgba(148, 163, 184, 0.3)",
};
