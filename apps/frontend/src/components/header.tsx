import { Link, useLocation } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { useAuth } from "@/contexts/auth-context";
import {
  Activity,
  Box,
  ChevronDown,
  HardDrive,
  HelpCircle,
  LogOut,
  type LucideIcon,
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
import { cn } from "@/lib/utils";

interface NavItem {
  readonly label: string;
  readonly href: string;
  readonly icon: LucideIcon;
  readonly excludedPaths?: readonly string[];
}

const NAV_ITEMS: readonly NavItem[] = [
  { label: "Servers", href: ROUTES.SERVERS, icon: Server },
  {
    label: "Containers",
    href: ROUTES.CONTAINERS,
    icon: Box,
    excludedPaths: [ROUTES.TEMPLATES],
  },
  { label: "Images", href: ROUTES.IMAGES, icon: HardDrive },
  { label: "Audit", href: ROUTES.AUDIT, icon: Activity },
];

function matchesPath(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(`${href}/`);
}

function isNavItemActive(pathname: string, item: NavItem): boolean {
  const isExcluded =
    item.excludedPaths?.some((path) => matchesPath(pathname, path)) ?? false;

  return !isExcluded && matchesPath(pathname, item.href);
}

export function Header() {
  const { user, isAuthenticated, logout } = useAuth();
  const { pathname } = useLocation();

  const displayName = user?.name ?? user?.githubLogin ?? "User";

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 pt-[env(safe-area-inset-top)]">
      <div className="container grid h-14 grid-cols-[minmax(max-content,1fr)_auto_minmax(max-content,1fr)] items-center gap-0.5 sm:h-16 sm:gap-4">
        <Link
          to={isAuthenticated ? ROUTES.HOME : ROUTES.LANDING}
          className="flex min-w-0 items-center gap-2 justify-self-start font-semibold text-lg"
          aria-label="flowDeploy - Go to home"
        >
          <Rocket className="h-6 w-6 shrink-0" aria-hidden="true" />
          <span className="hidden sm:inline">flowDeploy</span>
          <span className="hidden lg:inline-flex text-[10px] sm:text-xs font-medium px-1.5 py-0.5 rounded bg-primary/10 text-primary border border-primary/20">
            Self-hosted
          </span>
        </Link>

        {isAuthenticated ? (
          <nav
            className="flex h-full items-center justify-center gap-0.5 sm:gap-1"
            aria-label="Main navigation"
          >
            {NAV_ITEMS.map((item) => {
              const Icon = item.icon;
              const isActive = isNavItemActive(pathname, item);

              return (
                <Link
                  key={item.href}
                  to={item.href}
                  aria-label={item.label}
                  title={item.label}
                  aria-current={isActive ? "page" : undefined}
                  className={cn(
                    "inline-flex h-full min-w-11 items-center justify-center gap-2 border-b-2 px-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring xl:justify-start xl:px-3",
                    isActive
                      ? "border-primary text-foreground"
                      : "border-transparent text-muted-foreground hover:text-foreground",
                  )}
                >
                  <Icon className="h-4 w-4" aria-hidden="true" />
                  <span className="hidden lg:inline">{item.label}</span>
                </Link>
              );
            })}
          </nav>
        ) : null}

        <div className="flex items-center justify-self-end gap-0.5 sm:gap-2">
          {isAuthenticated && user ? (
            <>
              <Button
                asChild
                size="sm"
                className="h-11 w-11 px-0 sm:h-9 sm:w-auto sm:px-3"
              >
                <Link to={ROUTES.NEW_APP} aria-label="Create new application">
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  <span className="hidden lg:inline">New App</span>
                </Link>
              </Button>

              <span
                aria-hidden="true"
                className="hidden h-6 w-px bg-border sm:block"
              />

              <ThemeToggle />

              <span
                aria-hidden="true"
                className="hidden h-6 w-px bg-border sm:block"
              />

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    className="relative h-11 w-11 gap-1 rounded-full px-0 sm:h-9 sm:w-auto sm:px-2"
                    aria-label={`${displayName}'s account menu`}
                    aria-haspopup="menu"
                  >
                    <Avatar className="h-8 w-8">
                      <AvatarImage src={user.avatarUrl} alt="" />
                      <AvatarFallback>
                        <User className="h-4 w-4" aria-hidden="true" />
                      </AvatarFallback>
                    </Avatar>
                    <ChevronDown
                      className="hidden h-4 w-4 text-muted-foreground sm:block"
                      aria-hidden="true"
                    />
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
            <>
              <ThemeToggle />
              <Button asChild>
                <Link to={ROUTES.LOGIN} aria-label="Sign in to your account">
                  Sign in
                </Link>
              </Button>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
