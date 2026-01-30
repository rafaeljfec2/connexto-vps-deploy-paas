import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Stepper } from "@/components/ui/stepper";
import { DeployStep } from "./deploy-step";
import { EnvironmentStep } from "./environment-step";
import { RepositoryStep } from "./repository-step";
import { ReviewStep } from "./review-step";
import { INITIAL_DATA, ONBOARDING_STEPS, type OnboardingData } from "./types";

export function OnboardingWizard() {
  const navigate = useNavigate();
  const [currentStep, setCurrentStep] = useState(0);
  const [data, setData] = useState<OnboardingData>(INITIAL_DATA);

  const handleUpdate = (updates: Partial<OnboardingData>) => {
    setData((prev) => ({ ...prev, ...updates }));
  };

  const handleNext = () => {
    setCurrentStep((prev) => Math.min(prev + 1, ONBOARDING_STEPS.length - 1));
  };

  const handleBack = () => {
    setCurrentStep((prev) => Math.max(prev - 1, 0));
  };

  const stepProps = {
    data,
    onUpdate: handleUpdate,
    onNext: handleNext,
    onBack: handleBack,
  };

  const renderStep = () => {
    switch (currentStep) {
      case 0:
        return <RepositoryStep {...stepProps} />;
      case 1:
        return <EnvironmentStep {...stepProps} />;
      case 2:
        return <ReviewStep {...stepProps} />;
      case 3:
        return <DeployStep data={data} onBack={handleBack} />;
      default:
        return null;
    }
  };

  return (
    <div className="min-h-[calc(100vh-4rem)] flex flex-col">
      <div className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 sticky top-0 z-10">
        <div className="container max-w-3xl mx-auto px-4 py-4">
          <div className="flex items-center gap-4 mb-6">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => navigate("/")}
              className="shrink-0"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-lg font-semibold">Create New Application</h1>
              <p className="text-sm text-muted-foreground hidden md:block">
                Deploy your GitHub repository in minutes
              </p>
            </div>
          </div>

          <Stepper steps={ONBOARDING_STEPS} currentStep={currentStep} />
        </div>
      </div>

      <div className="flex-1 container max-w-3xl mx-auto px-4 py-6 md:py-8">
        {renderStep()}
      </div>
    </div>
  );
}
