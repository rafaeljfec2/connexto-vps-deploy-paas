import type { TroubleshootItem } from "@/features/servers/data/helper-server-setup";

interface TroubleshootSectionProps {
  readonly items: ReadonlyArray<TroubleshootItem>;
}

export function TroubleshootSection({ items }: TroubleshootSectionProps) {
  return (
    <div className="space-y-3">
      {items.map((item) => (
        <div
          key={item.problem}
          className="rounded-lg border bg-muted/20 p-3 sm:p-4 space-y-2"
        >
          <p className="font-medium text-sm break-words">{item.problem}</p>
          <p className="text-sm text-muted-foreground">
            <span className="font-medium">Cause:</span> {item.cause}
          </p>
          <p className="text-sm text-muted-foreground">
            <span className="font-medium">Solution:</span> {item.solution}
          </p>
        </div>
      ))}
    </div>
  );
}
