import { useState } from "react";
import { ChevronDown, ChevronRight, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";

interface JsonExpandableProps {
  readonly extra: Record<string, unknown>;
  readonly className?: string;
}

function formatValue(value: unknown): string {
  if (value === null) return "null";
  if (value === undefined) return "undefined";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  return JSON.stringify(value, null, 2);
}

function JsonKeyValue({
  name,
  value,
}: {
  readonly name: string;
  readonly value: unknown;
}) {
  const [copied, setCopied] = useState(false);
  const str = formatValue(value);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(str);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  return (
    <div className="flex items-start gap-2 py-0.5 group">
      <span className="text-amber-600 dark:text-amber-400 shrink-0 font-mono text-xs">
        {name}:
      </span>
      <span className="text-slate-400 font-mono text-xs break-all min-w-0 flex-1">
        {typeof value === "object" && value !== null ? (
          <pre className="whitespace-pre-wrap text-inherit">{str}</pre>
        ) : (
          str
        )}
      </span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-6 w-6 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
        onClick={handleCopy}
        aria-label="Copy value"
      >
        <Copy className={cn("h-3 w-3", copied && "text-green-400")} />
      </Button>
    </div>
  );
}

export function JsonExpandable({ extra, className }: JsonExpandableProps) {
  const [open, setOpen] = useState(false);
  const keys = Object.keys(extra);
  if (keys.length === 0) return null;

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div
        className={cn("pl-4 border-l-2 border-slate-700 ml-2 mt-1", className)}
      >
        <CollapsibleTrigger asChild>
          <button
            type="button"
            className="flex items-center gap-1 text-xs text-slate-500 hover:text-slate-400 font-mono"
            aria-expanded={open}
          >
            {open ? (
              <ChevronDown className="h-3 w-3 shrink-0" />
            ) : (
              <ChevronRight className="h-3 w-3 shrink-0" />
            )}
            <span>
              {keys.length} extra field{keys.length === 1 ? "" : "s"}
            </span>
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="pt-1 space-y-0">
            {keys.map((key) => (
              <JsonKeyValue key={key} name={key} value={extra[key]} />
            ))}
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
