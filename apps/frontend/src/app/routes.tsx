import { Routes, Route } from "react-router-dom";
import { DashboardPage } from "@/pages/dashboard";
import { AppDetailsPage } from "@/pages/app-details";
import { NewAppPage } from "@/pages/new-app";

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<DashboardPage />} />
      <Route path="/apps/new" element={<NewAppPage />} />
      <Route path="/apps/:id" element={<AppDetailsPage />} />
    </Routes>
  );
}
