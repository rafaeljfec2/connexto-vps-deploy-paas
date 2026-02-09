import { useEffect, useState } from "react";
import { Star } from "lucide-react";
import { useAnimateOnScroll } from "@/hooks/use-animate-on-scroll";

function GitHubIcon({ className }: { readonly className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
    </svg>
  );
}

function AnimatedCounter({
  target,
  suffix,
  isVisible,
}: {
  readonly target: number;
  readonly suffix: string;
  readonly isVisible: boolean;
}) {
  const [count, setCount] = useState(0);

  useEffect(() => {
    if (!isVisible) return;

    let current = 0;
    const increment = Math.ceil(target / 30);
    const interval = setInterval(() => {
      current = Math.min(current + increment, target);
      setCount(current);
      if (current >= target) {
        clearInterval(interval);
      }
    }, 40);

    return () => clearInterval(interval);
  }, [isVisible, target]);

  return (
    <span>
      {isVisible ? count : 0}
      {suffix}
    </span>
  );
}

export function SocialProofSection() {
  const techLogos = [
    "Docker",
    "GitHub",
    "Cloudflare",
    "Traefik",
    "PostgreSQL",
    "Go",
  ];

  const { ref: headerRef, isVisible: headerVisible } = useAnimateOnScroll();
  const { ref: statsRef, isVisible: statsVisible } = useAnimateOnScroll();
  const { ref: logosRef, isVisible: logosVisible } = useAnimateOnScroll();

  return (
    <section className="border-t border-border/40 bg-muted/30">
      <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-28 lg:px-8">
        <div
          ref={headerRef}
          className={`mx-auto max-w-2xl text-center transition-all duration-700 ${headerVisible ? "translate-y-0 opacity-100" : "translate-y-6 opacity-0"}`}
        >
          <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-4 py-1.5 text-sm font-medium text-emerald-600 dark:text-emerald-400">
            <GitHubIcon className="h-4 w-4" />
            <span>Open Source</span>
          </div>
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Trusted by developers
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Built in the open. Inspect the code, contribute, or self-host with
            full confidence.
          </p>
        </div>
        <div ref={statsRef} className="mt-12 grid gap-6 sm:grid-cols-3">
          <div
            className={`rounded-xl border border-border/50 bg-background p-6 text-center transition-all duration-600 ${statsVisible ? "translate-y-0 opacity-100" : "translate-y-8 opacity-0"}`}
            style={{ transitionDelay: "0ms" }}
          >
            <p className="text-3xl font-bold text-foreground">
              <AnimatedCounter
                target={100}
                suffix="+"
                isVisible={statsVisible}
              />
            </p>
            <p className="mt-1 text-sm text-muted-foreground">Deployments</p>
          </div>
          <div
            className={`rounded-xl border border-border/50 bg-background p-6 text-center transition-all duration-600 ${statsVisible ? "translate-y-0 opacity-100" : "translate-y-8 opacity-0"}`}
            style={{ transitionDelay: statsVisible ? "120ms" : "0ms" }}
          >
            <div className="flex items-center justify-center gap-1">
              <Star
                className={`h-5 w-5 text-yellow-500 transition-transform duration-500 ${statsVisible ? "rotate-0 scale-100" : "-rotate-45 scale-0"}`}
                aria-hidden="true"
              />
              <p className="text-3xl font-bold text-foreground">Open Source</p>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              Star us on GitHub
            </p>
          </div>
          <div
            className={`rounded-xl border border-border/50 bg-background p-6 text-center transition-all duration-600 ${statsVisible ? "translate-y-0 opacity-100" : "translate-y-8 opacity-0"}`}
            style={{ transitionDelay: statsVisible ? "240ms" : "0ms" }}
          >
            <p className="text-3xl font-bold text-foreground">
              <AnimatedCounter
                target={50}
                suffix="+"
                isVisible={statsVisible}
              />
            </p>
            <p className="mt-1 text-sm text-muted-foreground">
              Servers managed
            </p>
          </div>
        </div>
        <div
          ref={logosRef}
          className="mt-12 flex flex-wrap items-center justify-center gap-3"
        >
          {techLogos.map((name, index) => (
            <div
              key={name}
              className={`rounded-lg border border-border/50 bg-muted/30 px-4 py-2 text-sm text-muted-foreground transition-all hover:border-border hover:text-foreground ${logosVisible ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0"}`}
              style={{
                transitionDuration: "400ms",
                transitionDelay: logosVisible ? `${index * 60}ms` : "0ms",
              }}
            >
              {name}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
