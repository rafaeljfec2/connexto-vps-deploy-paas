import { Link } from "react-router-dom";
import { Plus, Rocket } from "lucide-react";
import { Button } from "@/components/ui/button";

export function Header() {
  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-14 items-center">
        <Link to="/" className="flex items-center gap-2 font-semibold">
          <Rocket className="h-5 w-5" />
          <span>PaaSDeploy</span>
        </Link>

        <nav className="ml-auto flex items-center gap-4">
          <Button asChild size="sm">
            <Link to="/apps/new">
              <Plus className="h-4 w-4" />
              New App
            </Link>
          </Button>
        </nav>
      </div>
    </header>
  );
}
