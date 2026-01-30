import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface ErrorMessageProps {
  readonly message: string;
  readonly className?: string;
  readonly variant?: "inline" | "block";
}

export function ErrorMessage({
  message,
  className,
  variant = "block",
}: ErrorMessageProps) {
  if (variant === "inline") {
    return (
      <div
        className={cn(
          "p-2 rounded bg-destructive/10 text-destructive text-sm",
          className,
        )}
      >
        {message}
      </div>
    );
  }

  return (
    <div className={cn("text-center py-12", className)}>
      <div className="rounded-full bg-destructive/10 p-4 mb-4 inline-flex">
        <AlertCircle className="h-8 w-8 text-destructive" />
      </div>
      <p className="text-destructive">{message}</p>
    </div>
  );
}
