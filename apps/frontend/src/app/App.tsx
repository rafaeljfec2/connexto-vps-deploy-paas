import { useEffect } from "react";
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
  useEffect(() => {
    document.documentElement.classList.add("dark");
  }, []);

  return (
    <Providers>
      <AppContent />
    </Providers>
  );
}
