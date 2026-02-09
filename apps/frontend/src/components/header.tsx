import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { useAuth } from "@/contexts/auth-context";
import {
  Activity,
  Box,
  HardDrive,
  HelpCircle,
  LogOut,
  Plus,
  Rocket,
  Server,
  Settings,
  User,
} from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ThemeToggle } from "@/components/theme-toggle";

export function Header() {
  const { user, isAuthenticated, logout } = useAuth();

  const displayName = user?.name ?? user?.githubLogin ?? "User";

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 pt-[env(safe-area-inset-top)]">
      <div className="container flex h-14 sm:h-16 items-center">
        <Link
          to={isAuthenticated ? ROUTES.HOME : ROUTES.LANDING}
          className="flex items-center gap-2 font-semibold text-lg"
          aria-label="FlowDeploy - Go to home"
        >
          <Rocket className="h-6 w-6" aria-hidden="true" />
          <span className="hidden sm:inline">FlowDeploy</span>
          <span className="text-[10px] sm:text-xs font-medium px-1.5 py-0.5 rounded bg-primary/10 text-primary border border-primary/20">
            Self-hosted
          </span>
        </Link>

        <nav
          className="ml-auto flex items-center gap-2"
          aria-label="Main navigation"
        >
          <ThemeToggle />

          {isAuthenticated && user ? (
            <>
              <Button asChild variant="ghost" size="sm">
                <Link to={ROUTES.CONTAINERS} aria-label="View containers">
                  <Box className="h-4 w-4 sm:mr-2" aria-hidden="true" />
                  <span className="hidden sm:inline">Containers</span>
                </Link>
              </Button>
              <Button asChild variant="ghost" size="sm">
                <Link to={ROUTES.IMAGES} aria-label="View Docker images">
                  <HardDrive className="h-4 w-4 sm:mr-2" aria-hidden="true" />
                  <span className="hidden sm:inline">Images</span>
                </Link>
              </Button>
              <Button asChild variant="ghost" size="sm">
                <Link to={ROUTES.AUDIT} aria-label="View audit logs">
                  <Activity className="h-4 w-4 sm:mr-2" aria-hidden="true" />
                  <span className="hidden sm:inline">Audit</span>
                </Link>
              </Button>

              <Button asChild variant="outline" size="sm">
                <Link to={ROUTES.NEW_APP} aria-label="Create new application">
                  <Plus className="h-4 w-4 sm:mr-2" aria-hidden="true" />
                  <span className="hidden sm:inline">New App</span>
                </Link>
              </Button>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    className="relative h-8 w-8 rounded-full"
                    aria-label={`${displayName}'s account menu`}
                    aria-haspopup="menu"
                  >
                    <Avatar className="h-8 w-8">
                      <AvatarImage src={user.avatarUrl} alt="" />
                      <AvatarFallback>
                        <User className="h-4 w-4" aria-hidden="true" />
                      </AvatarFallback>
                    </Avatar>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuLabel className="font-normal">
                    <div className="flex flex-col space-y-1">
                      <p className="text-sm font-medium leading-none truncate">
                        {displayName}
                      </p>
                      <p className="text-xs leading-none text-muted-foreground truncate">
                        {user.githubLogin ? `@${user.githubLogin}` : user.email}
                      </p>
                    </div>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem asChild className="cursor-pointer">
                    <Link to={ROUTES.IMAGES} role="menuitem">
                      <HardDrive className="mr-2 h-4 w-4" aria-hidden="true" />
                      Images
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild className="cursor-pointer">
                    <Link to={ROUTES.AUDIT} role="menuitem">
                      <Activity className="mr-2 h-4 w-4" aria-hidden="true" />
                      Audit Logs
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild className="cursor-pointer">
                    <Link to={ROUTES.SERVERS} role="menuitem">
                      <Server className="mr-2 h-4 w-4" aria-hidden="true" />
                      Servers
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild className="cursor-pointer">
                    <Link to={ROUTES.HELPER_SERVER_SETUP} role="menuitem">
                      <HelpCircle className="mr-2 h-4 w-4" aria-hidden="true" />
                      Helper
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild className="cursor-pointer">
                    <Link to={ROUTES.SETTINGS} role="menuitem">
                      <Settings className="mr-2 h-4 w-4" aria-hidden="true" />
                      Settings
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={logout}
                    className="cursor-pointer"
                    role="menuitem"
                  >
                    <LogOut className="mr-2 h-4 w-4" aria-hidden="true" />
                    Sign out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          ) : (
            <Button asChild>
              <Link to={ROUTES.LOGIN} aria-label="Sign in to your account">
                Sign in
              </Link>
            </Button>
          )}
        </nav>
      </div>
    </header>
  );
}
