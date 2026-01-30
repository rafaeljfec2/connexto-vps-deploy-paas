import { Route, Routes } from "react-router-dom";
import { AppDetailsPage } from "@/pages/app-details";
import { DashboardPage } from "@/pages/dashboard";
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
