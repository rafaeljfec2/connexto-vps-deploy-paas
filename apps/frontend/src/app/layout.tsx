import { Header } from "@/components/header";

interface LayoutProps {
  readonly children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="h-dvh bg-background flex flex-col overflow-hidden">
      <Header />
      <main className="container flex-1 min-h-0 overflow-auto py-4 sm:py-6 md:py-8 pb-[calc(1rem+env(safe-area-inset-bottom))]">
        {children}
      </main>
    </div>
  );
}
