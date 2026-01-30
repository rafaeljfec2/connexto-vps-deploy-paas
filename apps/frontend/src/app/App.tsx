import { useEffect } from "react";
import { Providers } from "./providers";
import { Layout } from "./layout";
import { AppRoutes } from "./routes";
import { useSSE } from "@/hooks/use-sse";

function AppContent() {
  useSSE();

  return (
    <Layout>
      <AppRoutes />
    </Layout>
  );
}

export function App() {
  useEffect(() => {
    document.documentElement.classList.add("dark");
  }, []);

  return (
    <Providers>
      <AppContent />
    </Providers>
  );
}
