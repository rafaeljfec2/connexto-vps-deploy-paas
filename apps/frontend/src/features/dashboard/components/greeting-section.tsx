import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { useAuth } from "@/contexts/auth-context";
import { Plus, Server } from "lucide-react";
import { Button } from "@/components/ui/button";

function getGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}

export function GreetingSection() {
  const { user } = useAuth();
  const firstName = user?.name?.split(" ")[0] ?? user?.githubLogin ?? "there";

  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
          {getGreeting()}, {firstName}
        </h1>
        <p className="text-sm text-muted-foreground sm:text-base">
          Here&apos;s what&apos;s happening with your infrastructure
        </p>
      </div>
      <div className="flex gap-2">
        <Button asChild variant="outline" size="sm">
          <Link to={ROUTES.SERVERS}>
            <Server className="mr-2 h-4 w-4" />
            Servers
          </Link>
        </Button>
        <Button asChild size="sm">
          <Link to={ROUTES.NEW_APP}>
            <Plus className="mr-2 h-4 w-4" />
            New App
          </Link>
        </Button>
      </div>
    </div>
  );
}
