import { Check, Minus, X } from "lucide-react";
import { useAnimateOnScroll } from "@/hooks/use-animate-on-scroll";

type ComparisonStatus = "yes" | "no" | "partial";

interface ComparisonRow {
  readonly feature: string;
  readonly vercel: ComparisonStatus;
  readonly coolify: ComparisonStatus;
  readonly flowdeploy: ComparisonStatus;
}

const COMPARISONS: readonly ComparisonRow[] = [
  { feature: "Self-hosted", vercel: "no", coolify: "yes", flowdeploy: "yes" },
  {
    feature: "Automatic DNS (Cloudflare)",
    vercel: "no",
    coolify: "no",
    flowdeploy: "yes",
  },
  {
    feature: "Multi-server support",
    vercel: "no",
    coolify: "partial",
    flowdeploy: "yes",
  },
  { feature: "Modern UI/UX", vercel: "yes", coolify: "no", flowdeploy: "yes" },
  {
    feature: "Monorepo support",
    vercel: "yes",
    coolify: "partial",
    flowdeploy: "yes",
  },
  { feature: "Free at scale", vercel: "no", coolify: "yes", flowdeploy: "yes" },
];

function StatusIcon({ status }: { readonly status: ComparisonStatus }) {
  if (status === "yes") {
    return (
      <div className="flex h-6 w-6 items-center justify-center rounded-full bg-emerald-500/10">
        <Check
          className="h-4 w-4 text-emerald-600 dark:text-emerald-400"
          aria-label="Yes"
        />
      </div>
    );
  }
  if (status === "no") {
    return (
      <div className="flex h-6 w-6 items-center justify-center rounded-full bg-destructive/10">
        <X className="h-4 w-4 text-destructive" aria-label="No" />
      </div>
    );
  }
  return (
    <div className="flex h-6 w-6 items-center justify-center rounded-full bg-yellow-500/10">
      <Minus
        className="h-4 w-4 text-yellow-600 dark:text-yellow-400"
        aria-label="Partial"
      />
    </div>
  );
}

export function ComparisonSection() {
  const { ref: headerRef, isVisible: headerVisible } = useAnimateOnScroll();
  const { ref: tableRef, isVisible: tableVisible } = useAnimateOnScroll({
    threshold: 0.1,
  });

  return (
    <section id="compare" className="border-t border-border/40">
      <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-28 lg:px-8">
        <div
          ref={headerRef}
          className={`mx-auto max-w-2xl text-center transition-all duration-700 ${headerVisible ? "translate-y-0 opacity-100" : "translate-y-6 opacity-0"}`}
        >
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Built different
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            See how FlowDeploy compares to other deployment platforms.
          </p>
        </div>
        <div
          ref={tableRef}
          className={`mx-auto mt-16 max-w-3xl overflow-x-auto rounded-xl border border-border/50 transition-all duration-700 ${tableVisible ? "scale-100 opacity-100" : "scale-95 opacity-0"}`}
        >
          <div className="grid grid-cols-4 gap-0 border-b border-border/50 bg-muted/50 px-4 py-3 text-sm font-medium text-foreground sm:px-6">
            <div>Feature</div>
            <div className="text-center">Vercel / Railway</div>
            <div className="text-center">Coolify</div>
            <div className="text-center font-semibold text-emerald-600 dark:text-emerald-400">
              FlowDeploy
            </div>
          </div>
          {COMPARISONS.map((row, index) => (
            <div
              key={row.feature}
              className={`grid grid-cols-4 gap-0 border-b border-border/30 px-4 py-3 text-sm last:border-b-0 sm:px-6 transition-all ${tableVisible ? "translate-x-0 opacity-100" : "translate-x-4 opacity-0"}`}
              style={{
                transitionDuration: "500ms",
                transitionDelay: tableVisible ? `${200 + index * 80}ms` : "0ms",
              }}
            >
              <div className="text-muted-foreground">{row.feature}</div>
              <div className="flex justify-center">
                <StatusIcon status={row.vercel} />
              </div>
              <div className="flex justify-center">
                <StatusIcon status={row.coolify} />
              </div>
              <div className="flex justify-center">
                <StatusIcon status={row.flowdeploy} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
