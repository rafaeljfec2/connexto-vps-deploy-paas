import { useCallback, useMemo, useState } from "react";
import type { ReactNode } from "react";
import type { ParsedLogEntry } from "../types";

function buildSearchRegex(search: string, useRegex: boolean): RegExp | null {
  const trimmed = search.trim();
  if (trimmed === "") return null;
  try {
    if (useRegex) {
      return new RegExp(trimmed, "gi");
    }
    const escaped = trimmed.replaceAll(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`);
    return new RegExp(escaped, "gi");
  } catch {
    return null;
  }
}

function entryMatchesSearch(
  entry: ParsedLogEntry,
  regex: RegExp | null,
): boolean {
  if (regex == null) return true;
  const searchable =
    entry.message +
    (entry.context ?? "") +
    JSON.stringify(entry.extra) +
    entry.raw;
  regex.lastIndex = 0;
  return regex.test(searchable);
}

export function useLogSearch(entries: readonly ParsedLogEntry[]) {
  const [searchTerm, setSearchTerm] = useState("");
  const [useRegex, setUseRegex] = useState(false);
  const [currentMatchIndex, setCurrentMatchIndex] = useState(0);

  const regex = useMemo(
    () => buildSearchRegex(searchTerm, useRegex),
    [searchTerm, useRegex],
  );

  const matchingEntries = useMemo(() => {
    if (regex == null) return [];
    return entries.filter((entry) => {
      regex.lastIndex = 0;
      return entryMatchesSearch(entry, regex);
    });
  }, [entries, regex]);

  const matchingLineNumbers = useMemo(
    () => matchingEntries.map((e) => e.lineNumber),
    [matchingEntries],
  );

  const currentMatchLine =
    matchingLineNumbers.length > 0
      ? matchingLineNumbers[
          ((currentMatchIndex % matchingLineNumbers.length) +
            matchingLineNumbers.length) %
            matchingLineNumbers.length
        ]
      : undefined;

  const goToNext = useCallback(() => {
    if (matchingLineNumbers.length === 0) return;
    setCurrentMatchIndex((i) => (i + 1) % matchingLineNumbers.length);
  }, [matchingLineNumbers.length]);

  const goToPrevious = useCallback(() => {
    if (matchingLineNumbers.length === 0) return;
    setCurrentMatchIndex(
      (i) => (i - 1 + matchingLineNumbers.length) % matchingLineNumbers.length,
    );
  }, [matchingLineNumbers.length]);

  const goToMatch = useCallback((index: number) => {
    setCurrentMatchIndex(index);
  }, []);

  const clearSearch = useCallback(() => {
    setSearchTerm("");
    setCurrentMatchIndex(0);
  }, []);

  return {
    searchTerm,
    setSearchTerm,
    useRegex,
    setUseRegex,
    regex,
    matchingLineNumbers,
    currentMatchIndex,
    currentMatchLine,
    matchCount: matchingLineNumbers.length,
    goToNext,
    goToPrevious,
    goToMatch,
    clearSearch,
    hasSearch: searchTerm.trim().length > 0,
  };
}

export function highlightText(
  text: string,
  searchTerm: string,
  regex: RegExp | null,
): ReactNode {
  if (searchTerm.trim() === "" || regex == null) return text;
  regex.lastIndex = 0;
  const match = regex.exec(text);
  if (match == null) return text;
  const parts: ReactNode[] = [];
  let lastIndex = 0;
  regex.lastIndex = 0;
  let m: RegExpExecArray | null;
  while ((m = regex.exec(text)) !== null) {
    if (m.index > lastIndex) {
      parts.push(text.slice(lastIndex, m.index));
    }
    parts.push(
      <mark
        key={`${m.index}-${m[0]}`}
        className="bg-yellow-500/40 text-inherit rounded px-0.5"
      >
        {m[0]}
      </mark>,
    );
    lastIndex = regex.lastIndex;
  }
  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }
  return parts.length === 0 ? text : <>{parts}</>;
}
