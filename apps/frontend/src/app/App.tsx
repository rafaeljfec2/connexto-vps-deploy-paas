import { useSSE } from "@/hooks/use-sse";
import { Layout } from "./layout";
import { Providers } from "./providers";
import { AppRoutes } from "./routes";

function AppContent() {
  useSSE();

  return (
    <Layout>
      <AppRoutes />
    </Layout>
  );
}

export function App() {
  return (
    <Providers>
      <AppContent />
    </Providers>
  );
}
