import {
  Activity,
  FolderGit2,
  Globe,
  RotateCcw,
  Server,
  Shield,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useAnimateOnScroll } from "@/hooks/use-animate-on-scroll";

interface FeatureItem {
  readonly icon: LucideIcon;
  readonly title: string;
  readonly description: string;
}

const FEATURES: readonly FeatureItem[] = [
  {
    icon: Shield,
    title: "Zero-downtime deploys",
    description:
      "Automatic health checks and instant rollback if something goes wrong.",
  },
  {
    icon: Globe,
    title: "Your domains, automated",
    description:
      "Cloudflare DNS and SSL configured automatically. No manual nginx configs.",
  },
  {
    icon: Activity,
    title: "Real-time visibility",
    description:
      "Live logs, CPU/memory metrics, and container console. Know exactly what is happening.",
  },
  {
    icon: Server,
    title: "Multi-server ready",
    description:
      "Deploy across multiple servers with agent-based architecture. Scale without limits.",
  },
  {
    icon: FolderGit2,
    title: "Monorepo native",
    description:
      "Deploy specific apps from monorepos. Point to a directory, flowDeploy handles the rest.",
  },
  {
    icon: RotateCcw,
    title: "One-click rollback",
    description:
      "Something broke? Roll back to any previous version in one click.",
  },
];

export function FeaturesSection() {
  const { ref: headerRef, isVisible: headerVisible } = useAnimateOnScroll();
  const { ref: gridRef, isVisible: gridVisible } = useAnimateOnScroll();

  return (
    <section id="features" className="border-t border-border/40 bg-muted/30">
      <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-28 lg:px-8">
        <div
          ref={headerRef}
          className={`mx-auto max-w-2xl text-center transition-all duration-700 ${headerVisible ? "translate-y-0 opacity-100" : "translate-y-6 opacity-0"}`}
        >
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Everything you need to ship with confidence
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Built for developers who want full control without the complexity.
          </p>
        </div>
        <div
          ref={gridRef}
          className="mt-16 grid gap-6 sm:grid-cols-2 lg:grid-cols-3"
        >
          {FEATURES.map((feature, index) => {
            const Icon = feature.icon;
            return (
              <div
                key={feature.title}
                className={`group rounded-xl border border-border/50 bg-background p-6 transition-all hover:border-emerald-500/30 hover:-translate-y-1 hover:shadow-md ${gridVisible ? "translate-y-0 opacity-100" : "translate-y-8 opacity-0"}`}
                style={{
                  transitionDuration: "600ms",
                  transitionDelay: gridVisible ? `${index * 100}ms` : "0ms",
                }}
              >
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-500/10 transition-colors group-hover:bg-emerald-500/20">
                  <Icon
                    className="h-5 w-5 text-emerald-600 dark:text-emerald-400 transition-transform duration-300 group-hover:scale-110"
                    aria-hidden="true"
                  />
                </div>
                <h3 className="mt-4 text-lg font-semibold text-foreground">
                  {feature.title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                  {feature.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
