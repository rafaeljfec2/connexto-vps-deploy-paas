import { Header } from "@/components/header";

interface LayoutProps {
  readonly children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-dvh bg-background flex flex-col">
      <Header />
      <main className="container flex-1 py-6 sm:py-8 safe-bottom safe-x">
        {children}
      </main>
    </div>
  );
}
