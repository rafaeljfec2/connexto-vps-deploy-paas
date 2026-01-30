import { Check, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface Step {
  readonly id: string;
  readonly title: string;
  readonly icon: LucideIcon;
}

interface StepperProps {
  readonly steps: readonly Step[];
  readonly currentStep: number;
  readonly className?: string;
}

export function Stepper({ steps, currentStep, className }: StepperProps) {
  return (
    <nav aria-label="Progress" className={className}>
      <ol className="flex items-center w-full">
        {steps.map((step, index) => {
          const isCompleted = index < currentStep;
          const isCurrent = index === currentStep;
          const isLast = index === steps.length - 1;
          const Icon = step.icon;

          return (
            <li
              key={step.id}
              className={cn("flex items-center", !isLast && "flex-1")}
            >
              <div className="flex flex-col items-center gap-2">
                <div
                  className={cn(
                    "flex h-10 w-10 items-center justify-center rounded-full border-2 transition-colors",
                    isCompleted &&
                      "border-primary bg-primary text-primary-foreground",
                    isCurrent && "border-primary bg-primary/10 text-primary",
                    !isCompleted &&
                      !isCurrent &&
                      "border-muted-foreground/30 bg-background text-muted-foreground",
                  )}
                >
                  {isCompleted ? (
                    <Check className="h-5 w-5" />
                  ) : (
                    <Icon className="h-5 w-5" />
                  )}
                </div>
                <span
                  className={cn(
                    "text-xs font-medium whitespace-nowrap",
                    isCurrent ? "text-foreground" : "text-muted-foreground",
                  )}
                >
                  {step.title}
                </span>
              </div>

              {!isLast && (
                <div
                  className={cn(
                    "h-0.5 flex-1 mx-3 mt-[-1.5rem] transition-colors",
                    isCompleted ? "bg-primary" : "bg-muted-foreground/30",
                  )}
                />
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
