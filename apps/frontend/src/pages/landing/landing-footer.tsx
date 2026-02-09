import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { Rocket } from "lucide-react";
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

export function LandingFooter() {
  const currentYear = new Date().getFullYear();
  const { ref, isVisible } = useAnimateOnScroll({ threshold: 0.2 });

  return (
    <footer ref={ref} className="border-t border-border/40">
      <div
        className={`mx-auto max-w-6xl px-4 py-12 transition-all duration-700 sm:px-6 lg:px-8 ${isVisible ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0"}`}
      >
        <div className="flex flex-col items-center gap-6 sm:flex-row sm:justify-between">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Rocket className="h-4 w-4" aria-hidden="true" />
            <span>FlowDeploy</span>
          </div>
          <nav
            className="flex items-center gap-6 text-sm text-muted-foreground"
            aria-label="Footer"
          >
            <a
              href="https://github.com"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 transition-colors hover:text-foreground"
            >
              <GitHubIcon className="h-4 w-4" />
              GitHub
            </a>
            <Link
              to={ROUTES.TERMS}
              className="transition-colors hover:text-foreground"
            >
              Terms
            </Link>
          </nav>
        </div>
        <div className="mt-8 border-t border-border/30 pt-6 text-center text-xs text-muted-foreground">
          <p>{currentYear} FlowDeploy. Built with Go and React.</p>
        </div>
      </div>
    </footer>
  );
}
