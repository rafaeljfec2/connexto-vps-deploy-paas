import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { useAuth } from "@/contexts/auth-context";
import {
  Activity,
  Box,
  Command,
  Plus,
  Rocket,
  Server,
  Terminal,
  Zap,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useDashboardStats } from "../hooks/use-dashboard-stats";
import {
  buildHeroSubtitle,
  buildNextBestAction,
  pickNorthStar,
} from "../lib/hero-insights";
import { type KpiKey, deriveKpiTones } from "../lib/kpi-tone";
import { HeroNextBestAction } from "./hero-next-best-action";
import { KpiChip } from "./kpi-chip";
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

interface KpiDefinition {
  readonly key: KpiKey;
  readonly icon: typeof Rocket;
  readonly label: string;
  readonly value: string | number;
  readonly subtitle: string;
}

export function HeroDashboard() {
  const { user } = useAuth();
  const { toggle: toggleCommandPalette } = useCommandPalette();
  const stats = useDashboardStats();

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

  const kpiDefinitions: readonly KpiDefinition[] = [
    {
      key: "apps",
      icon: Rocket,
      label: "Apps",
      value: totalApps,
      subtitle: `${totalApps} total`,
    },
    {
      key: "servers",
      icon: Server,
      label: "Servers",
      value: `${onlineServers}/${totalServers}`,
      subtitle: "online",
    },
    {
      key: "containers",
      icon: Box,
      label: "Containers",
      value: `${runningContainers}/${totalContainers}`,
      subtitle: "running",
    },
    {
      key: "deploys",
      icon: Zap,
      label: "Deploys",
      value: deployTotal,
      subtitle: deploySubtitle,
    },
  ];

  const primaryKpi = kpiDefinitions.find((k) => k.key === northStar);
  const secondaryKpis = kpiDefinitions.filter((k) => k.key !== northStar);

  return (
    <section
      aria-labelledby="hero-dashboard-title"
      className="relative overflow-hidden rounded-2xl border border-border/60 bg-card"
    >
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-60 dark:opacity-30"
        style={{
          backgroundImage:
            "radial-gradient(circle, hsl(var(--muted-foreground) / 0.07) 1px, transparent 1px)",
          backgroundSize: "24px 24px",
        }}
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-32 -top-32 h-[420px] w-[420px] rounded-full bg-emerald-400/10 blur-3xl animate-glow-pulse dark:bg-emerald-500/10"
      />

      <div className="relative grid gap-6 px-4 py-8 sm:px-6 sm:py-10 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)] lg:px-8 lg:py-12">
        <div className="min-w-0">
          <HeroNextBestAction action={nextBestAction} />

          <div className="mb-4 flex flex-wrap items-center gap-2">
            <div className="inline-flex items-center gap-2 rounded-full border border-border/60 bg-muted/50 px-3 py-1 text-xs text-muted-foreground opacity-0 animate-fade-in-up">
              <Terminal className="h-3.5 w-3.5" aria-hidden="true" />
              <span>
                {isLoading
                  ? "Loading infrastructure pulse…"
                  : `${totalApps} apps · ${onlineServers}/${totalServers} servers online`}
              </span>
            </div>
            <button
              type="button"
              onClick={toggleCommandPalette}
              className="inline-flex items-center gap-1.5 rounded-full border border-border/60 bg-muted/30 px-2.5 py-1 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
              aria-label="Open command palette"
            >
              <Command className="h-3 w-3" aria-hidden="true" />
              <span>K</span>
            </button>
          </div>

          <h1
            id="hero-dashboard-title"
            className="text-3xl font-bold tracking-tight text-foreground opacity-0 animate-fade-in-up [animation-delay:0.1s] sm:text-4xl lg:text-5xl"
          >
            {getGreeting()},{" "}
            <span className="text-emerald-600 dark:text-emerald-400">
              {firstName}
            </span>
            .
          </h1>
          <p className="mt-3 max-w-2xl text-sm text-muted-foreground opacity-0 animate-fade-in-up [animation-delay:0.2s] sm:text-base">
            {subtitle}
          </p>

          <div className="mt-6 flex flex-wrap gap-2 opacity-0 animate-fade-in-up [animation-delay:0.3s]">
            {primaryKpi && (
              <KpiChip
                key={primaryKpi.key}
                icon={primaryKpi.icon}
                label={primaryKpi.label}
                value={primaryKpi.value}
                subtitle={primaryKpi.subtitle}
                tone={tones[primaryKpi.key]}
                emphasis="primary"
                isLoading={isLoading}
              />
            )}
            {secondaryKpis.map((kpi) => (
              <KpiChip
                key={kpi.key}
                icon={kpi.icon}
                label={kpi.label}
                value={kpi.value}
                subtitle={kpi.subtitle}
                tone={tones[kpi.key]}
                isLoading={isLoading}
              />
            ))}
          </div>

          <div className="mt-7 flex flex-col gap-2 opacity-0 animate-fade-in-up [animation-delay:0.4s] sm:flex-row sm:flex-wrap sm:items-center">
            <Button asChild size="sm" className="h-10 px-4">
              <Link to={ROUTES.NEW_APP}>
                <Plus className="mr-2 h-4 w-4" aria-hidden="true" />
                New App
              </Link>
            </Button>
            <Button asChild variant="outline" size="sm" className="h-10 px-4">
              <Link to={ROUTES.SERVERS}>
                <Server className="mr-2 h-4 w-4" aria-hidden="true" />
                Servers
              </Link>
            </Button>
            <Button asChild variant="ghost" size="sm" className="h-10 px-4">
              <a href="#activity-feed">
                <Activity className="mr-2 h-4 w-4" aria-hidden="true" />
                Activity
              </a>
            </Button>
          </div>
        </div>

        <aside className="min-w-0 opacity-0 animate-fade-in-up [animation-delay:0.5s]">
          <LiveActivityPanel
            recentDeploys={recentDeploys}
            isLoading={isLoading}
          />
        </aside>
      </div>
    </section>
  );
}
