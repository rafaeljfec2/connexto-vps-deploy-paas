interface CodeBlockProps {
  readonly children: string;
}

export function CodeBlock({ children }: CodeBlockProps) {
  return (
    <pre className="overflow-x-auto rounded-md border bg-muted/50 p-3 text-sm">
      <code>{children}</code>
    </pre>
  );
}
