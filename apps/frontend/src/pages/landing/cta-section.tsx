import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAnimateOnScroll } from "@/hooks/use-animate-on-scroll";

export function CtaSection() {
  const { ref, isVisible } = useAnimateOnScroll({ threshold: 0.2 });

  return (
    <section className="border-t border-border/40">
      <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-28 lg:px-8">
        <div
          ref={ref}
          className={`relative overflow-hidden rounded-2xl border border-border/50 bg-muted/20 px-6 py-16 text-center transition-all duration-700 sm:px-12 sm:py-20 ${isVisible ? "scale-100 opacity-100" : "scale-95 opacity-0"}`}
        >
          <div
            className="pointer-events-none absolute -left-20 -top-20 h-[300px] w-[300px] rounded-full bg-emerald-400/10 blur-3xl animate-glow-pulse dark:bg-emerald-500/10"
            aria-hidden="true"
          />
          <div
            className="pointer-events-none absolute -bottom-20 -right-20 h-[300px] w-[300px] rounded-full bg-blue-400/10 blur-3xl animate-glow-pulse [animation-delay:2s] dark:bg-blue-500/5"
            aria-hidden="true"
          />
          <div className="relative">
            <h2
              className={`text-3xl font-bold tracking-tight text-foreground transition-all duration-700 sm:text-4xl ${isVisible ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0"}`}
              style={{ transitionDelay: isVisible ? "200ms" : "0ms" }}
            >
              Stop managing deploys. Start shipping.
            </h2>
            <p
              className={`mx-auto mt-4 max-w-xl text-lg text-muted-foreground transition-all duration-700 ${isVisible ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0"}`}
              style={{ transitionDelay: isVisible ? "350ms" : "0ms" }}
            >
              Set up FlowDeploy on your server in under 5 minutes. Free, open
              source, forever.
            </p>
            <div
              className={`mt-8 flex flex-col items-center gap-4 transition-all duration-700 sm:flex-row sm:justify-center ${isVisible ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0"}`}
              style={{ transitionDelay: isVisible ? "500ms" : "0ms" }}
            >
              <Button
                asChild
                size="lg"
                className="h-12 px-8 text-base transition-transform hover:scale-[1.02] active:scale-[0.98]"
              >
                <Link to={ROUTES.REGISTER}>
                  Get Started
                  <ArrowRight className="ml-2 h-4 w-4" aria-hidden="true" />
                </Link>
              </Button>
            </div>
            <p
              className={`mt-4 text-sm text-muted-foreground transition-all duration-700 ${isVisible ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0"}`}
              style={{ transitionDelay: isVisible ? "600ms" : "0ms" }}
            >
              No credit card required. Self-hosted on your infrastructure.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}
