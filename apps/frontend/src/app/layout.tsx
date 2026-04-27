import { CommandPalette } from "@/components/command-palette";
import { Header } from "@/components/header";
import { CommandPaletteProvider } from "@/hooks/use-command-palette";

interface LayoutProps {
  readonly children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <CommandPaletteProvider>
      <div className="min-h-dvh bg-background flex flex-col">
        <Header />
        <main className="container flex-1 py-4 sm:py-6 md:py-8 pb-[calc(1rem+env(safe-area-inset-bottom))]">
          {children}
        </main>
      </div>
      <CommandPalette />
    </CommandPaletteProvider>
  );
}
