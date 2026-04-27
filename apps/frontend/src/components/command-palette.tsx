import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import {
  Activity,
  Box,
  FileText,
  Image,
  LayoutDashboard,
  Moon,
  Plus,
  Server,
  Settings,
  Sun,
} from "lucide-react";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "@/components/ui/command";
import { DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useTheme } from "@/hooks/use-theme";

interface NavigationEntry {
  readonly id: string;
  readonly label: string;
  readonly href: string;
  readonly icon: typeof LayoutDashboard;
  readonly keywords?: readonly string[];
}

const NAVIGATION: readonly NavigationEntry[] = [
  {
    id: "nav-dashboard",
    label: "Dashboard",
    href: ROUTES.HOME,
    icon: LayoutDashboard,
  },
  {
    id: "nav-new-app",
    label: "New App",
    href: ROUTES.NEW_APP,
    icon: Plus,
    keywords: ["create", "deploy"],
  },
  { id: "nav-servers", label: "Servers", href: ROUTES.SERVERS, icon: Server },
  {
    id: "nav-containers",
    label: "Containers",
    href: ROUTES.CONTAINERS,
    icon: Box,
  },
  {
    id: "nav-templates",
    label: "Templates",
    href: ROUTES.TEMPLATES,
    icon: FileText,
  },
  { id: "nav-images", label: "Images", href: ROUTES.IMAGES, icon: Image },
  { id: "nav-audit", label: "Audit log", href: ROUTES.AUDIT, icon: Activity },
  {
    id: "nav-settings",
    label: "Settings",
    href: ROUTES.SETTINGS,
    icon: Settings,
  },
];

export function CommandPalette() {
  const { open, setOpen } = useCommandPalette();
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();

  const runCommand = useCallback(
    (command: () => void) => {
      setOpen(false);
      command();
    },
    [setOpen],
  );

  const nextTheme = theme === "dark" ? "light" : "dark";

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <DialogTitle className="sr-only">Command palette</DialogTitle>
      <DialogDescription className="sr-only">
        Navigate the app or run an action. Use arrow keys to move, enter to
        select, escape to close.
      </DialogDescription>
      <CommandInput placeholder="Search commands, pages, actions…" />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        <CommandGroup heading="Navigate">
          {NAVIGATION.map((entry) => {
            const Icon = entry.icon;
            return (
              <CommandItem
                key={entry.id}
                value={`${entry.label} ${entry.keywords?.join(" ") ?? ""}`}
                onSelect={() => runCommand(() => navigate(entry.href))}
              >
                <Icon className="mr-2 h-4 w-4" aria-hidden="true" />
                <span>{entry.label}</span>
              </CommandItem>
            );
          })}
        </CommandGroup>

        <CommandSeparator />

        <CommandGroup heading="Actions">
          <CommandItem
            value={`Toggle theme ${nextTheme}`}
            onSelect={() => runCommand(() => setTheme(nextTheme))}
          >
            {theme === "dark" ? (
              <Sun className="mr-2 h-4 w-4" aria-hidden="true" />
            ) : (
              <Moon className="mr-2 h-4 w-4" aria-hidden="true" />
            )}
            <span>Switch to {nextTheme} theme</span>
            <CommandShortcut>⌘⇧L</CommandShortcut>
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
