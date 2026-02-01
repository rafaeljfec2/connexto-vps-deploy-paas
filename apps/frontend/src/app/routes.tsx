import { Route, Routes } from "react-router-dom";
import { AppDetailsPage } from "@/pages/app-details";
import { ContainersPage } from "@/pages/containers";
import { DashboardPage } from "@/pages/dashboard";
import { LoginPage } from "@/pages/login";
import { MigrationPage } from "@/pages/migration";
import { NewAppPage } from "@/pages/new-app";
import { SettingsPage } from "@/pages/settings";
import { TemplatesPage } from "@/pages/templates";
import { ProtectedRoute } from "@/components/protected-route";

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
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
    </Routes>
  );
}
