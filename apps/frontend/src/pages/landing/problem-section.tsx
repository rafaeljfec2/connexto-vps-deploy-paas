import { AlertTriangle, DollarSign, Monitor } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useAnimateOnScroll } from "@/hooks/use-animate-on-scroll";

interface ProblemItem {
  readonly icon: LucideIcon;
  readonly title: string;
  readonly description: string;
}

const PROBLEMS: readonly ProblemItem[] = [
  {
    icon: AlertTriangle,
    title: "Manual deploys are fragile",
    description:
      "SSH into servers, run scripts, pray nothing breaks. Every deploy is a risk.",
  },
  {
    icon: DollarSign,
    title: "PaaS platforms get expensive fast",
    description:
      "Start free, then pay per app per month. At 10 apps, you spend more than your VPS.",
  },
  {
    icon: Monitor,
    title: "Self-hosted tools feel outdated",
    description:
      "Clunky UIs, missing features, zero developer experience. You deserve better.",
  },
];

export function ProblemSection() {
  const { ref: headerRef, isVisible: headerVisible } = useAnimateOnScroll();
  const { ref: cardsRef, isVisible: cardsVisible } = useAnimateOnScroll();

  return (
    <section className="border-t border-border/40 bg-muted/30">
      <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-28 lg:px-8">
        <div
          ref={headerRef}
          className={`mx-auto max-w-2xl text-center transition-all duration-700 ${headerVisible ? "translate-y-0 opacity-100" : "translate-y-6 opacity-0"}`}
        >
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            You should not need a DevOps team to deploy a web app
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Deploying should be simple. But the current options make it harder
            than it needs to be.
          </p>
        </div>

        <div
          ref={cardsRef}
          className="mt-16 grid gap-8 sm:grid-cols-2 lg:grid-cols-3"
        >
          {PROBLEMS.map((problem, index) => {
            const Icon = problem.icon;
            return (
              <div
                key={problem.title}
                className={`rounded-xl border border-border/50 bg-background p-6 transition-all hover:border-border hover:-translate-y-1 hover:shadow-md ${cardsVisible ? "translate-y-0 opacity-100" : "translate-y-8 opacity-0"}`}
                style={{
                  transitionDuration: "600ms",
                  transitionDelay: cardsVisible ? `${index * 120}ms` : "0ms",
                }}
              >
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-destructive/10">
                  <Icon
                    className="h-5 w-5 text-destructive"
                    aria-hidden="true"
                  />
                </div>
                <h3 className="mt-4 text-lg font-semibold text-foreground">
                  {problem.title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                  {problem.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
