interface StepNumberProps {
  readonly n: number;
}

export function StepNumber({ n }: StepNumberProps) {
  return (
    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">
      {n}
    </span>
  );
}
