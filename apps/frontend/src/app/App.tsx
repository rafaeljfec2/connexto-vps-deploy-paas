import { useSSE } from "@/hooks/use-sse";
import { Providers } from "./providers";
import { AppRoutes } from "./routes";

function AppContent() {
  useSSE();
  return <AppRoutes />;
}

export function App() {
  return (
    <Providers>
      <AppContent />
    </Providers>
  );
}
