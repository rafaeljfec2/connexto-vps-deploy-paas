import { useState } from "react";
import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { Menu, Rocket, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ThemeToggle } from "@/components/theme-toggle";

const NAV_LINKS = [
  { label: "Features", href: "#features" },
  { label: "How it works", href: "#how-it-works" },
  { label: "Compare", href: "#compare" },
] as const;

export function LandingHeader() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 opacity-0 animate-fade-in backdrop-blur-lg">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <Link
          to={ROUTES.LANDING}
          className="flex items-center gap-2 font-semibold text-lg"
          aria-label="flowDeploy - Home"
        >
          <Rocket className="h-6 w-6" aria-hidden="true" />
          <span>flowDeploy</span>
        </Link>

        <nav
          className="hidden items-center gap-1 md:flex"
          aria-label="Main navigation"
        >
          {NAV_LINKS.map((link) => (
            <a
              key={link.href}
              href={link.href}
              className="rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              {link.label}
            </a>
          ))}
        </nav>

        <div className="flex items-center gap-2">
          <ThemeToggle />

          <div className="hidden items-center gap-2 md:flex">
            <Button asChild variant="ghost" size="sm">
              <Link to={ROUTES.LOGIN}>Sign in</Link>
            </Button>
            <Button asChild size="sm">
              <Link to={ROUTES.REGISTER}>Get Started</Link>
            </Button>
          </div>

          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            aria-label="Toggle menu"
            aria-expanded={mobileMenuOpen}
          >
            {mobileMenuOpen ? (
              <X className="h-5 w-5" />
            ) : (
              <Menu className="h-5 w-5" />
            )}
          </Button>
        </div>
      </div>

      <div
        className={`overflow-hidden border-t border-border/40 bg-background transition-all duration-300 ease-in-out md:hidden ${mobileMenuOpen ? "max-h-80 opacity-100" : "max-h-0 border-t-0 opacity-0"}`}
      >
        <nav
          className="flex flex-col gap-2 px-4 py-4"
          aria-label="Mobile navigation"
        >
          {NAV_LINKS.map((link) => (
            <a
              key={link.href}
              href={link.href}
              className="rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
              onClick={() => setMobileMenuOpen(false)}
            >
              {link.label}
            </a>
          ))}
          <div className="mt-2 flex flex-col gap-2 border-t border-border/40 pt-4">
            <Button asChild variant="outline" className="w-full">
              <Link to={ROUTES.LOGIN}>Sign in</Link>
            </Button>
            <Button asChild className="w-full">
              <Link to={ROUTES.REGISTER}>Get Started</Link>
            </Button>
          </div>
        </nav>
      </div>
    </header>
  );
}
