import { Header } from "@/components/header";

interface LayoutProps {
  readonly children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-screen bg-background flex flex-col">
      <Header />
      <main className="container flex-1 py-8">{children}</main>
    </div>
  );
}
