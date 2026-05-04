import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { useAuth } from "@/contexts/auth-context";
import { Activity, Command, Plus, Server, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { HeroBackground } from "@/components/hero-background";
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useSSEConnectionStatus } from "@/hooks/use-sse";
import { cn } from "@/lib/utils";
import { useDashboardStats } from "../hooks/use-dashboard-stats";
import {
  buildHeroSubtitle,
  buildNextBestAction,
  pickNorthStar,
} from "../lib/hero-insights";
import { type KpiKey, type KpiTone, deriveKpiTones } from "../lib/kpi-tone";
import { HeroNextBestAction } from "./hero-next-best-action";
import { LiveActivityPanel } from "./live-activity-panel";

function getGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}

function resolveFirstName(
  name: string | null | undefined,
  githubLogin: string | null | undefined,
): string {
  const trimmed = name?.trim() ?? "";
  if (trimmed.length > 0) {
    const [first] = trimmed.split(" ");
    if (first) return first;
  }
  const login = githubLogin?.trim() ?? "";
  return login.length > 0 ? login : "there";
}

interface StatDefinition {
  readonly key: KpiKey;
  readonly label: string;
  readonly value: string;
  readonly subtitle: string;
}

const TONE_TEXT: Record<KpiTone, string> = {
  default: "text-foreground",
  success: "text-emerald-600 dark:text-emerald-400",
  warning: "text-yellow-600 dark:text-yellow-400",
  destructive: "text-red-600 dark:text-red-400",
};

const TONE_DOT: Record<KpiTone, string> = {
  default: "bg-muted-foreground/40",
  success: "bg-emerald-500",
  warning: "bg-yellow-500",
  destructive: "bg-red-500",
};

export function HeroDashboard() {
  const { user } = useAuth();
  const { toggle: toggleCommandPalette } = useCommandPalette();
  const stats = useDashboardStats();
  const isSSEConnected = useSSEConnectionStatus();

  const {
    totalApps,
    totalServers,
    onlineServers,
    runningContainers,
    totalContainers,
    successfulDeploys,
    failedDeploys,
    recentDeploys,
    isLoading,
  } = stats;

  const firstName = resolveFirstName(user?.name, user?.githubLogin);
  const tones = deriveKpiTones(stats);
  const northStar = pickNorthStar(tones);
  const nextBestAction = isLoading
    ? null
    : buildNextBestAction({
        tones,
        onlineServers,
        totalServers,
        runningContainers,
        totalContainers,
        successfulDeploys,
        failedDeploys,
      });
  const subtitle = isLoading
    ? "Syncing infrastructure status…"
    : buildHeroSubtitle(recentDeploys, totalApps, totalServers);

  const deployTotal = successfulDeploys + failedDeploys;
  const deploySubtitle =
    failedDeploys > 0
      ? `${successfulDeploys} ok · ${failedDeploys} fail`
      : deployTotal > 0
        ? `${successfulDeploys} ok`
        : "no runs yet";

  const stat: readonly StatDefinition[] = [
    {
      key: "apps",
      label: "Apps",
      value: String(totalApps),
      subtitle: totalApps === 1 ? "1 total" : `${totalApps} total`,
    },
    {
      key: "servers",
      label: "Servers",
      value: `${onlineServers}/${totalServers}`,
      subtitle: "online",
    },
    {
      key: "containers",
      label: "Containers",
      value: `${runningContainers}/${totalContainers}`,
      subtitle: "running",
    },
    {
      key: "deploys",
      label: "Deploys",
      value: String(deployTotal),
      subtitle: deploySubtitle,
    },
  ];

  return (
    <section
      aria-labelledby="hero-dashboard-title"
      className="relative overflow-hidden rounded-xl border border-border/60 bg-card"
    >
      <HeroBackground
        glowClassName="h-[280px] w-[280px] -right-20 -top-20 bg-emerald-400/5 dark:bg-emerald-500/5"
        dotOpacityClassName="opacity-40 dark:opacity-20"
      />

      <div className="relative grid gap-5 px-4 py-5 sm:px-6 sm:py-6 lg:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)] lg:gap-8 lg:px-7 lg:py-7">
        <div className="min-w-0">
          <HeroNextBestAction action={nextBestAction} />

          <div className="mb-3 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1.5">
              <Terminal className="h-3 w-3" aria-hidden="true" />
              <span>
                {isLoading
                  ? "Loading infrastructure pulse…"
                  : `${totalApps} apps · ${onlineServers}/${totalServers} servers online`}
              </span>
            </span>
            <span aria-hidden="true" className="text-border">
              ·
            </span>
            <button
              type="button"
              onClick={toggleCommandPalette}
              className="inline-flex items-center gap-1 rounded-md border border-border/60 bg-muted/30 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
              aria-label="Open command palette"
            >
              <Command className="h-3 w-3" aria-hidden="true" />
              <span>K</span>
            </button>
          </div>

          <h1
            id="hero-dashboard-title"
            className="text-2xl font-semibold tracking-tight text-foreground opacity-0 animate-fade-in-up [animation-delay:0.1s] sm:text-3xl lg:text-4xl"
          >
            {getGreeting()},{" "}
            <span className="text-emerald-600 dark:text-emerald-400">
              {firstName}
            </span>
            .
          </h1>
          <p className="mt-1.5 max-w-2xl text-sm text-muted-foreground opacity-0 animate-fade-in-up [animation-delay:0.2s]">
            {subtitle}
          </p>

          <dl className="mt-5 grid grid-cols-2 gap-x-6 gap-y-3 border-t border-border/50 pt-4 opacity-0 animate-fade-in-up [animation-delay:0.3s] sm:grid-cols-4">
            {stat.map((item) => {
              const tone = tones[item.key];
              const isNorthStar = item.key === northStar && tone !== "default";
              return (
                <div key={item.key} className="min-w-0">
                  <dt className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                    {isNorthStar && (
                      <span
                        className={cn(
                          "h-1.5 w-1.5 rounded-full",
                          TONE_DOT[tone],
                        )}
                        aria-hidden="true"
                      />
                    )}
                    <span className="truncate">{item.label}</span>
                  </dt>
                  {isLoading ? (
                    <dd aria-busy="true" className="mt-1.5">
                      <Skeleton className="h-5 w-14" />
                      <Skeleton className="mt-1 h-2.5 w-10" />
                    </dd>
                  ) : (
                    <dd
                      className={cn(
                        "mt-1 flex items-baseline gap-1.5 truncate",
                        TONE_TEXT[tone],
                      )}
                    >
                      <span className="text-lg font-semibold tracking-tight sm:text-xl">
                        {item.value}
                      </span>
                      <span className="truncate text-[11px] text-muted-foreground">
                        {item.subtitle}
                      </span>
                    </dd>
                  )}
                </div>
              );
            })}
          </dl>

          <div className="mt-5 flex flex-wrap items-center gap-1.5 border-t border-border/50 pt-4 opacity-0 animate-fade-in-up [animation-delay:0.4s]">
            <Button asChild size="sm" className="h-9">
              <Link to={ROUTES.NEW_APP}>
                <Plus className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                New App
              </Link>
            </Button>
            <Button asChild variant="outline" size="sm" className="h-9">
              <Link to={ROUTES.SERVERS}>
                <Server className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                Servers
              </Link>
            </Button>
            <Button asChild variant="ghost" size="sm" className="h-9">
              <a href="#activity-feed">
                <Activity className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                Activity
              </a>
            </Button>
          </div>
        </div>

        <aside className="min-w-0 opacity-0 animate-fade-in-up [animation-delay:0.5s]">
          <LiveActivityPanel
            recentDeploys={recentDeploys}
            isLoading={isLoading}
            isSSEConnected={isSSEConnected}
          />
        </aside>
      </div>
    </section>
  );
}
