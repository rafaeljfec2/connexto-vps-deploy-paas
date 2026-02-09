import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { ArrowRight, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
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

interface TerminalLine {
  readonly text: string;
  readonly prefix?: string;
  readonly prefixClass?: string;
  readonly lineClass: string;
  readonly isLast?: boolean;
}

const TERMINAL_LINES: readonly TerminalLine[] = [
  {
    text: "git push origin main",
    prefix: "$ ",
    prefixClass: "text-emerald-400",
    lineClass: "text-white/90",
  },
  { text: "Enumerating objects: 12, done.", lineClass: "text-white/50" },
  {
    text: "Compressing objects: 100% (8/8), done.",
    lineClass: "text-white/50",
  },
  {
    text: "Build started...",
    prefix: "FlowDeploy ",
    prefixClass: "text-emerald-400",
    lineClass: "text-white/70 mt-3",
  },
  {
    text: "Building container...",
    prefix: "FlowDeploy ",
    prefixClass: "text-emerald-400",
    lineClass: "text-white/70",
  },
  {
    text: "Health check passed",
    prefix: "FlowDeploy ",
    prefixClass: "text-emerald-400",
    lineClass: "text-white/70",
  },
  {
    text: "SSL certificate configured",
    prefix: "FlowDeploy ",
    prefixClass: "text-emerald-400",
    lineClass: "text-white/70",
  },
  {
    text: "Deployed to app.yourdomain.com",
    prefix: "Ready ",
    prefixClass: "text-emerald-400",
    lineClass: "mt-1 font-semibold text-white",
    isLast: true,
  },
];

function TerminalMockup() {
  const { ref, isVisible } = useAnimateOnScroll({ threshold: 0.3 });
  const [visibleLines, setVisibleLines] = useState(0);

  useEffect(() => {
    if (!isVisible) return;

    let current = 0;
    const interval = setInterval(() => {
      current += 1;
      setVisibleLines(current);
      if (current >= TERMINAL_LINES.length) {
        clearInterval(interval);
      }
    }, 280);

    return () => clearInterval(interval);
  }, [isVisible]);

  return (
    <div
      ref={ref}
      className="relative mx-auto w-full max-w-2xl overflow-hidden rounded-xl border border-border/60 bg-slate-950 shadow-2xl opacity-0 animate-scale-in [animation-delay:0.6s]"
    >
      <div className="flex items-center gap-2 border-b border-white/10 px-4 py-3">
        <div className="flex gap-1.5">
          <div className="h-3 w-3 rounded-full bg-red-500/80" />
          <div className="h-3 w-3 rounded-full bg-yellow-500/80" />
          <div className="h-3 w-3 rounded-full bg-green-500/80" />
        </div>
        <span className="ml-2 text-xs font-mono text-white/40">terminal</span>
      </div>
      <div className="space-y-1.5 p-4 font-mono text-sm leading-relaxed">
        {TERMINAL_LINES.map((line, i) => (
          <p
            key={line.text}
            className={`${line.lineClass} transition-all duration-300 ${i < visibleLines ? "translate-y-0 opacity-100" : "translate-y-1 opacity-0"}`}
          >
            {line.prefix && (
              <span className={line.prefixClass}>{line.prefix}</span>
            )}
            {line.isLast ? (
              <span>
                {"Deployed to "}
                <span className="text-emerald-400 underline decoration-emerald-400/30">
                  app.yourdomain.com
                </span>
              </span>
            ) : (
              <span>{line.text}</span>
            )}
          </p>
        ))}
        {visibleLines >= TERMINAL_LINES.length && (
          <span className="mt-1 inline-block h-4 w-2 animate-pulse bg-emerald-400/80" />
        )}
      </div>
    </div>
  );
}

export function HeroSection() {
  return (
    <section className="relative overflow-hidden">
      <div
        className="pointer-events-none absolute inset-0 opacity-60 dark:opacity-30"
        aria-hidden="true"
        style={{
          backgroundImage:
            "radial-gradient(circle, hsl(var(--muted-foreground) / 0.07) 1px, transparent 1px)",
          backgroundSize: "24px 24px",
        }}
      />
      <div
        className="pointer-events-none absolute -right-40 -top-40 h-[500px] w-[500px] rounded-full bg-emerald-400/10 blur-3xl animate-glow-pulse dark:bg-emerald-500/10"
        aria-hidden="true"
      />

      <div className="relative mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-28 lg:px-8 lg:py-36">
        <div className="mx-auto max-w-3xl text-center">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-border/60 bg-muted/50 px-4 py-1.5 text-sm text-muted-foreground opacity-0 animate-fade-in-up">
            <Terminal className="h-4 w-4" aria-hidden="true" />
            <span>Open Source - Self-hosted PaaS</span>
          </div>

          <h1 className="text-4xl font-bold tracking-tight text-foreground opacity-0 animate-fade-in-up [animation-delay:0.1s] sm:text-5xl md:text-6xl lg:text-7xl">
            Deploy from Git to your own server.{" "}
            <span className="text-emerald-600 dark:text-emerald-400">
              Automatically.
            </span>
          </h1>

          <p className="mx-auto mt-6 max-w-2xl text-lg text-muted-foreground opacity-0 animate-fade-in-up [animation-delay:0.2s] sm:text-xl">
            FlowDeploy gives you Vercel-like deploys on your own infrastructure.
            Push to main, get a live URL. No vendor lock-in, no surprise bills.
          </p>

          <div className="mt-10 flex flex-col items-center gap-4 opacity-0 animate-fade-in-up [animation-delay:0.35s] sm:flex-row sm:justify-center">
            <Button
              asChild
              size="lg"
              className="h-12 px-8 text-base transition-transform hover:scale-[1.02] active:scale-[0.98]"
            >
              <Link to={ROUTES.REGISTER}>
                Get Started
                <ArrowRight
                  className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-0.5"
                  aria-hidden="true"
                />
              </Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="lg"
              className="h-12 px-8 text-base transition-transform hover:scale-[1.02] active:scale-[0.98]"
            >
              <a
                href="https://github.com"
                target="_blank"
                rel="noopener noreferrer"
              >
                <GitHubIcon className="mr-2 h-4 w-4" />
                View on GitHub
              </a>
            </Button>
          </div>
        </div>

        <div className="mt-16 sm:mt-20 lg:mt-24">
          <TerminalMockup />
        </div>
      </div>
    </section>
  );
}
