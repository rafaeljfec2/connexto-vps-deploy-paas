import { cn } from "@/lib/utils";

interface FormFieldProps {
  readonly label: string;
  readonly htmlFor: string;
  readonly helper?: string;
  readonly error?: string;
  readonly required?: boolean;
  readonly children: React.ReactNode;
  readonly className?: string;
}

export function FormField({
  label,
  htmlFor,
  helper,
  error,
  required,
  children,
  className,
}: FormFieldProps) {
  return (
    <div className={cn("space-y-2", className)}>
      <label htmlFor={htmlFor} className="text-sm font-medium">
        {label}
        {required && <span className="text-destructive ml-1">*</span>}
      </label>
      {children}
      {helper && !error && (
        <p className="text-xs text-muted-foreground">{helper}</p>
      )}
      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  );
}
