import { Suspense, lazy } from "react";
import { Outlet, Route, Routes } from "react-router-dom";
import { Layout } from "@/app/layout";
import { Loader2 } from "lucide-react";
import { ProtectedRoute } from "@/components/protected-route";

const LandingPage = lazy(() =>
  import("@/pages/landing/landing-page").then((m) => ({
    default: m.LandingPage,
  })),
);
const LoginPage = lazy(() =>
  import("@/pages/login").then((m) => ({ default: m.LoginPage })),
);
const RegisterPage = lazy(() =>
  import("@/pages/register").then((m) => ({ default: m.RegisterPage })),
);
const TermsOfServicePage = lazy(() =>
  import("@/pages/terms-of-service").then((m) => ({
    default: m.TermsOfServicePage,
  })),
);
const DashboardPage = lazy(() =>
  import("@/pages/dashboard").then((m) => ({ default: m.DashboardPage })),
);
const NewAppPage = lazy(() =>
  import("@/pages/new-app").then((m) => ({ default: m.NewAppPage })),
);
const AppDetailsPage = lazy(() =>
  import("@/pages/app-details").then((m) => ({ default: m.AppDetailsPage })),
);
const SettingsPage = lazy(() =>
  import("@/pages/settings").then((m) => ({ default: m.SettingsPage })),
);
const ServersPage = lazy(() =>
  import("@/pages/servers").then((m) => ({ default: m.ServersPage })),
);
const ServerDetailsPage = lazy(() =>
  import("@/pages/server-details").then((m) => ({
    default: m.ServerDetailsPage,
  })),
);
const HelperServerSetupPage = lazy(() =>
  import("@/pages/helper-server-setup").then((m) => ({
    default: m.HelperServerSetupPage,
  })),
);
const MigrationPage = lazy(() =>
  import("@/pages/migration").then((m) => ({ default: m.MigrationPage })),
);
const ContainersPage = lazy(() =>
  import("@/pages/containers").then((m) => ({ default: m.ContainersPage })),
);
const TemplatesPage = lazy(() =>
  import("@/pages/templates").then((m) => ({ default: m.TemplatesPage })),
);
const ImagesPage = lazy(() =>
  import("@/pages/images").then((m) => ({ default: m.ImagesPage })),
);
const AuditPage = lazy(() =>
  import("@/pages/audit").then((m) => ({ default: m.AuditPage })),
);

function PageFallback() {
  return (
    <div className="flex h-[50vh] items-center justify-center">
      <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
    </div>
  );
}

function AppLayout() {
  return (
    <Layout>
      <Suspense fallback={<PageFallback />}>
        <Outlet />
      </Suspense>
    </Layout>
  );
}

export function AppRoutes() {
  return (
    <Suspense fallback={<PageFallback />}>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/terms" element={<TermsOfServicePage />} />

        <Route element={<AppLayout />}>
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <DashboardPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/apps/new"
            element={
              <ProtectedRoute>
                <NewAppPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/apps/:id"
            element={
              <ProtectedRoute>
                <AppDetailsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/settings"
            element={
              <ProtectedRoute>
                <SettingsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/servers"
            element={
              <ProtectedRoute>
                <ServersPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/servers/:id"
            element={
              <ProtectedRoute>
                <ServerDetailsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/helper/server-setup"
            element={
              <ProtectedRoute>
                <HelperServerSetupPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/settings/migration"
            element={
              <ProtectedRoute>
                <MigrationPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/containers"
            element={
              <ProtectedRoute>
                <ContainersPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/containers/templates"
            element={
              <ProtectedRoute>
                <TemplatesPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/images"
            element={
              <ProtectedRoute>
                <ImagesPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/audit"
            element={
              <ProtectedRoute>
                <AuditPage />
              </ProtectedRoute>
            }
          />
        </Route>
      </Routes>
    </Suspense>
  );
}
