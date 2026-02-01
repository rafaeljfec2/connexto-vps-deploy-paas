import { Header } from "@/components/header";

interface LayoutProps {
  readonly children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-dvh bg-background flex flex-col">
      <Header />
      <main className="container flex-1 py-4 sm:py-6 md:py-8 pb-[calc(1rem+env(safe-area-inset-bottom))]">
        {children}
      </main>
    </div>
  );
}
