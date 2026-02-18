import { useCallback, useState } from "react";

const RESET_DELAY_MS = 2000;

export function useCopyToClipboard() {
  const [copied, setCopied] = useState(false);

  const copy = useCallback(async (text: string): Promise<boolean> => {
    if (!text) return false;

    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), RESET_DELAY_MS);
      return true;
    } catch (error) {
      console.error("Failed to copy to clipboard", error);
      return false;
    }
  }, []);

  return { copy, copied };
}
