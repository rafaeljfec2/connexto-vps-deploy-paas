import { GitBranch, Globe, Server } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useAnimateOnScroll } from "@/hooks/use-animate-on-scroll";

interface StepItem {
  readonly step: number;
  readonly icon: LucideIcon;
  readonly title: string;
  readonly description: string;
}

const STEPS: readonly StepItem[] = [
  {
    step: 1,
    icon: Server,
    title: "Connect your server",
    description:
      "Add your VPS with one command. FlowDeploy installs everything automatically.",
  },
  {
    step: 2,
    icon: GitBranch,
    title: "Link your GitHub repo",
    description:
      "Select a repository, choose a branch, set your environment variables.",
  },
  {
    step: 3,
    icon: Globe,
    title: "Push and deploy",
    description:
      "Every push to main triggers a build. SSL, domains, and routing handled automatically.",
  },
];

function StepCard({
  item,
  index,
}: {
  readonly item: StepItem;
  readonly index: number;
}) {
  const { ref, isVisible } = useAnimateOnScroll({ threshold: 0.2 });
  const Icon = item.icon;

  return (
    <div
      ref={ref}
      className={`relative flex flex-col gap-6 transition-all duration-700 lg:flex-row lg:items-center lg:gap-12 ${isVisible ? "translate-x-0 opacity-100" : "-translate-x-8 opacity-0"}`}
      style={{ transitionDelay: isVisible ? `${index * 150}ms` : "0ms" }}
    >
      <div className="flex shrink-0 items-center gap-4 lg:w-16">
        <div
          className={`relative z-10 flex h-16 w-16 items-center justify-center rounded-full border-2 border-emerald-500/30 bg-emerald-500/10 transition-all duration-500 ${isVisible ? "scale-100" : "scale-75"}`}
        >
          <span className="text-lg font-bold text-emerald-600 dark:text-emerald-400">
            {item.step}
          </span>
        </div>
      </div>
      <div className="flex-1 rounded-xl border border-border/50 bg-muted/20 p-6 transition-all hover:border-border hover:shadow-sm sm:p-8">
        <div className="flex items-center gap-3">
          <Icon
            className="h-5 w-5 text-emerald-600 dark:text-emerald-400"
            aria-hidden="true"
          />
          <h3 className="text-xl font-semibold text-foreground">
            {item.title}
          </h3>
        </div>
        <p className="mt-3 leading-relaxed text-muted-foreground">
          {item.description}
        </p>
      </div>
    </div>
  );
}

export function SolutionSection() {
  const { ref: headerRef, isVisible: headerVisible } = useAnimateOnScroll();

  return (
    <section id="how-it-works" className="border-t border-border/40">
      <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-28 lg:px-8">
        <div
          ref={headerRef}
          className={`mx-auto max-w-2xl text-center transition-all duration-700 ${headerVisible ? "translate-y-0 opacity-100" : "translate-y-6 opacity-0"}`}
        >
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            From git push to production in seconds
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Three steps to automated deployments on your own infrastructure.
          </p>
        </div>
        <div className="relative mt-16">
          <div
            className="absolute left-8 top-0 hidden h-full w-px bg-border lg:block"
            aria-hidden="true"
          />
          <div className="space-y-12 lg:space-y-16">
            {STEPS.map((item, index) => (
              <StepCard key={item.step} item={item} index={index} />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
