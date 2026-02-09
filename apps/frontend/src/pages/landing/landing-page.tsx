import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { useAuth } from "@/contexts/auth-context";
import { ComparisonSection } from "./comparison-section";
import { CtaSection } from "./cta-section";
import { FeaturesSection } from "./features-section";
import { HeroSection } from "./hero-section";
import { LandingFooter } from "./landing-footer";
import { LandingHeader } from "./landing-header";
import { ProblemSection } from "./problem-section";
import { SocialProofSection } from "./social-proof-section";
import { SolutionSection } from "./solution-section";

export function LandingPage() {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      navigate(ROUTES.HOME, { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading) {
    return (
      <div className="flex h-dvh items-center justify-center">
        <div
          className="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"
          aria-hidden="true"
        />
      </div>
    );
  }

  return (
    <div className="min-h-dvh bg-background">
      <LandingHeader />
      <main>
        <HeroSection />
        <ProblemSection />
        <SolutionSection />
        <FeaturesSection />
        <ComparisonSection />
        <SocialProofSection />
        <CtaSection />
      </main>
      <LandingFooter />
    </div>
  );
}
