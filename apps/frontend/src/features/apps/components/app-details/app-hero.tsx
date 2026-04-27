import { Link } from "react-router-dom";
import {
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  ChevronsDownUp,
  ChevronsUpDown,
  ExternalLink,
  Folder,
  GitBranch,
  Globe,
  RefreshCw,
  RotateCcw,
  Terminal,
} from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { HealthIndicator } from "@/components/health-indicator";
import { HeroBackground } from "@/components/hero-background";
import { StatusBadge } from "@/components/status-badge";
import { AppSettingsDialog } from "@/features/apps/components/app-settings-dialog";
import type { useAppActions } from "@/features/apps/hooks/use-app-actions";
import type { KpiTone } from "@/features/dashboard/lib/kpi-tone";
import { cn, formatRepositoryUrl } from "@/lib/utils";
import type { App, ContainerStats, Deployment, HealthStatus } from "@/types";
import {
  type AppKpiKey,
  type AppNbaAction,
  type AppNextBestAction,
  buildAppNextBestAction,
  buildAppSubtitle,
  deriveAppKpiTones,
  parseHealthState,
  pickAppNorthStar,
} from "../../lib/app-hero-insights";
import { AppRecentDeploysPanel } from "./app-recent-deploys-panel";

interface AppHeroProps {
  readonly app: App;
  readonly health: HealthStatus | null | undefined;
  readonly deployments: readonly Deployment[] | undefined;
  readonly containerStats: ContainerStats | null | undefined;
  readonly openAppUrl: string | null;
  readonly actions: ReturnType<typeof useAppActions>;
  readonly allExpanded: boolean;
  readonly toggleAllSections: () => void;
  readonly hasSuccessfulDeploy: boolean;
}

interface StatDefinition {
  readonly key: AppKpiKey;
  readonly label: string;
  readonly value: string;
}

const ANCHOR_BY_ACTION: Record<AppNbaAction, string> = {
  "view-logs": "#container-logs-section",
  "view-health": "#container-health-section",
  "view-metrics": "#resource-metrics-section",
  redeploy: "#deployments-section",
};

const HEALTH_LABEL: Record<ReturnType<typeof parseHealthState>, string> = {
  healthy: "Healthy",
  unhealthy: "Unhealthy",
  starting: "Starting",
  offline: "Offline",
  unknown: "Unknown",
};

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

function formatPercent(percent: number | null | undefined): string {
  if (percent == null || Number.isNaN(percent)) return "—";
  if (percent < 10) return `${percent.toFixed(1)}%`;
  return `${Math.round(percent)}%`;
}

function deployLabelOf(latestDeploy: Deployment | null): string {
  if (!latestDeploy) return "—";
  switch (latestDeploy.status) {
    case "success":
      return "Success";
    case "failed":
      return "Failed";
    case "running":
      return "Running";
    case "pending":
      return "Pending";
    default:
      return "Cancelled";
  }
}

interface StatsInput {
  readonly latestDeploy: Deployment | null;
  readonly healthState: ReturnType<typeof parseHealthState>;
  readonly containerStats: ContainerStats | null | undefined;
}

function buildStatDefinitions(input: StatsInput): readonly StatDefinition[] {
  const hasStats = input.containerStats != null;
  const base: StatDefinition[] = [
    {
      key: "lastDeploy",
      label: "Last deploy",
      value: deployLabelOf(input.latestDeploy),
    },
    {
      key: "health",
      label: "Health",
      value: HEALTH_LABEL[input.healthState],
    },
  ];
  if (hasStats) {
    base.push(
      {
        key: "cpu",
        label: "CPU",
        value: formatPercent(input.containerStats?.cpuPercent),
      },
      {
        key: "memory",
        label: "Memory",
        value: formatPercent(input.containerStats?.memoryPercent),
      },
    );
  }
  return base;
}

function NbaBanner({
  action,
  onRedeploy,
}: {
  readonly action: AppNextBestAction | null;
  readonly onRedeploy: () => void;
}) {
  if (!action) return null;
  const isDestructive = action.severity === "destructive";
  const title = isDestructive ? "Action required" : "Needs your attention";
  const href = ANCHOR_BY_ACTION[action.action];

  const content = (
    <>
      {action.ctaLabel}
      <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
    </>
  );

  return (
    <Alert
      variant={isDestructive ? "destructive" : "default"}
      className={cn(
        "mb-4 flex items-start gap-3",
        !isDestructive &&
          "border-yellow-500/40 bg-yellow-500/5 text-yellow-700 dark:border-yellow-500/30 dark:text-yellow-300",
      )}
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div className="flex-1 space-y-0.5">
        <AlertTitle className="text-sm font-semibold">{title}</AlertTitle>
        <AlertDescription className="text-xs">
          {action.message}
        </AlertDescription>
      </div>
      {action.action === "redeploy" ? (
        <button
          type="button"
          onClick={onRedeploy}
          className="inline-flex shrink-0 items-center gap-1 self-center text-xs font-medium underline-offset-4 hover:underline"
        >
          {content}
        </button>
      ) : (
        <a
          href={href}
          className="inline-flex shrink-0 items-center gap-1 self-center text-xs font-medium underline-offset-4 hover:underline"
        >
          {content}
        </a>
      )}
    </Alert>
  );
}

export function AppHero({
  app,
  health,
  deployments,
  containerStats,
  openAppUrl,
  actions,
  allExpanded,
  toggleAllSections,
  hasSuccessfulDeploy,
}: AppHeroProps) {
  const latestDeploy = deployments?.[0] ?? null;
  const healthState = parseHealthState(health);
  const tones = deriveAppKpiTones({
    latestDeploy,
    healthState,
    containerStats,
  });
  const northStar = pickAppNorthStar(tones);
  const nba = buildAppNextBestAction({ latestDeploy, healthState, tones });
  const subtitle = buildAppSubtitle({
    latestDeploy,
    openAppUrl,
    branch: app.branch,
  });
  const stats = buildStatDefinitions({
    latestDeploy,
    healthState,
    containerStats,
  });
  const showWorkdir = app.workdir && app.workdir !== ".";

  return (
    <section
      aria-labelledby="app-hero-title"
      className="relative overflow-hidden rounded-xl border border-border/60 bg-card"
    >
      <HeroBackground
        glowClassName="h-[260px] w-[260px] -right-16 -top-16 bg-emerald-400/5 dark:bg-emerald-500/5"
        dotOpacityClassName="opacity-40 dark:opacity-20"
      />

      <div className="relative grid gap-5 px-4 py-5 sm:px-6 sm:py-6 lg:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)] lg:gap-8 lg:px-7 lg:py-7">
        <div className="min-w-0">
          <div className="mb-3 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
            <Button
              asChild
              variant="ghost"
              size="sm"
              className="-ml-2 h-7 gap-1.5 px-2 text-muted-foreground hover:text-foreground"
            >
              <Link to="/dashboard">
                <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
                <span>Dashboard</span>
              </Link>
            </Button>
            <span aria-hidden="true">/</span>
            <span>Apps</span>
            <span aria-hidden="true">/</span>
            <span className="truncate font-medium text-foreground">
              {app.name}
            </span>
          </div>

          <NbaBanner action={nba} onRedeploy={() => actions.handleRedeploy()} />

          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <h1
              id="app-hero-title"
              className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl"
            >
              {app.name}
            </h1>
            <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1.5">
                <HealthIndicator health={health} size="sm" />
                <span className={cn("font-medium", TONE_TEXT[tones.health])}>
                  {HEALTH_LABEL[healthState]}
                </span>
              </span>
              {latestDeploy && (
                <>
                  <span aria-hidden="true" className="text-border">
                    ·
                  </span>
                  <StatusBadge status={latestDeploy.status} size="sm" />
                </>
              )}
              {app.appVersion && (
                <>
                  <span aria-hidden="true" className="text-border">
                    ·
                  </span>
                  <span className="font-mono">v{app.appVersion}</span>
                </>
              )}
            </div>
          </div>

          <p className="mt-1.5 max-w-2xl text-sm text-muted-foreground">
            {subtitle}
          </p>

          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1.5">
              <GitBranch className="h-3 w-3" aria-hidden="true" />
              {app.branch}
            </span>
            <span aria-hidden="true" className="text-border">
              ·
            </span>
            <a
              href={app.repositoryUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex max-w-[220px] items-center gap-1 truncate hover:text-foreground sm:max-w-none"
            >
              <ExternalLink className="h-3 w-3 shrink-0" aria-hidden="true" />
              <span className="truncate">
                {formatRepositoryUrl(app.repositoryUrl)}
              </span>
            </a>
            {showWorkdir && (
              <>
                <span aria-hidden="true" className="text-border">
                  ·
                </span>
                <span className="inline-flex items-center gap-1.5 font-mono">
                  <Folder className="h-3 w-3" aria-hidden="true" />
                  {app.workdir}
                </span>
              </>
            )}
          </div>

          <dl className="mt-5 grid grid-cols-2 gap-x-6 gap-y-3 border-t border-border/50 pt-4 sm:grid-cols-4">
            {stats.map((stat) => {
              const tone = tones[stat.key];
              const isNorthStar = stat.key === northStar && tone !== "default";
              return (
                <div key={stat.key} className="min-w-0">
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
                    <span className="truncate">{stat.label}</span>
                  </dt>
                  <dd
                    className={cn(
                      "mt-1 truncate text-base font-semibold tracking-tight",
                      TONE_TEXT[tone],
                    )}
                  >
                    {stat.value}
                  </dd>
                </div>
              );
            })}
          </dl>

          <div className="mt-5 flex flex-wrap items-center gap-1.5 border-t border-border/50 pt-4">
            <Button
              size="sm"
              onClick={() => actions.handleRedeploy()}
              disabled={actions.redeploy.isPending}
              className="h-9"
            >
              <RefreshCw
                className={cn(
                  "mr-1.5 h-3.5 w-3.5",
                  actions.redeploy.isPending && "animate-spin",
                )}
                aria-hidden="true"
              />
              Redeploy
            </Button>
            {openAppUrl && (
              <Button asChild size="sm" variant="outline" className="h-9">
                <a href={openAppUrl} target="_blank" rel="noopener noreferrer">
                  <Globe className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                  Open App
                </a>
              </Button>
            )}
            <Button
              size="sm"
              variant="outline"
              onClick={actions.handleRollback}
              disabled={actions.rollback.isPending || !hasSuccessfulDeploy}
              className="h-9"
            >
              <RotateCcw className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              Rollback
            </Button>
            <span
              aria-hidden="true"
              className="mx-1 hidden h-5 w-px bg-border sm:inline-block"
            />
            <Button asChild size="sm" variant="ghost" className="h-9">
              <a href="#container-logs-section">
                <Terminal className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                Logs
              </a>
            </Button>
            <AppSettingsDialog app={app} />
            <Button
              size="sm"
              variant="ghost"
              onClick={toggleAllSections}
              className="hidden h-9 sm:inline-flex"
            >
              {allExpanded ? (
                <ChevronsDownUp
                  className="mr-1.5 h-3.5 w-3.5"
                  aria-hidden="true"
                />
              ) : (
                <ChevronsUpDown
                  className="mr-1.5 h-3.5 w-3.5"
                  aria-hidden="true"
                />
              )}
              {allExpanded ? "Collapse" : "Expand"}
            </Button>
          </div>
        </div>

        <aside className="min-w-0">
          <AppRecentDeploysPanel
            deployments={deployments}
            onRedeploy={() => actions.handleRedeploy()}
          />
        </aside>
      </div>
    </section>
  );
}
