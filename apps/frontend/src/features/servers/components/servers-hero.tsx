import { Link } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { ROUTES } from "@/constants/routes";
import {
  ArrowLeft,
  ArrowRight,
  HelpCircle,
  Plus,
  RefreshCw,
  Server as ServerIcon,
  Terminal,
  TriangleAlert,
} from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { HeroBackground } from "@/components/hero-background";
import { cn } from "@/lib/utils";
import type { Server } from "@/types";
import { SERVERS_QUERY_KEY } from "../hooks/use-servers";
import {
  type ServersNextBestAction,
  type ServersSummary,
  buildServersNextBestAction,
  buildServersSubtitle,
  summarizeServers,
} from "../lib/servers-insights";
import { AddServerDialog } from "./add-server-dialog";

interface ServersHeroProps {
  readonly servers: readonly Server[] | undefined;
  readonly isLoading: boolean;
}

type StatTone = "default" | "success" | "warning" | "destructive";

const TONE_TEXT: Record<StatTone, string> = {
  default: "text-foreground",
  success: "text-emerald-600 dark:text-emerald-400",
  warning: "text-yellow-600 dark:text-yellow-400",
  destructive: "text-red-600 dark:text-red-400",
};

const TONE_DOT: Record<StatTone, string> = {
  default: "bg-muted-foreground/40",
  success: "bg-emerald-500",
  warning: "bg-yellow-500",
  destructive: "bg-red-500",
};

interface StatItem {
  readonly key: string;
  readonly label: string;
  readonly value: string;
  readonly hint: string;
  readonly tone: StatTone;
  readonly isNorthStar?: boolean;
}

function buildStats(summary: ServersSummary): readonly StatItem[] {
  const offlineTone: StatTone =
    summary.error > 0
      ? "destructive"
      : summary.offline > 0 || summary.pending > 0
        ? "warning"
        : "default";

  const onlineTone: StatTone =
    summary.total === 0
      ? "default"
      : summary.online === summary.total
        ? "success"
        : summary.online === 0
          ? "destructive"
          : "warning";

  const agentsTone: StatTone =
    summary.outdatedAgents > 0 ? "warning" : "default";

  const unhealthy = summary.error + summary.offline + summary.pending;

  let northStarKey: string = "online";
  if (summary.error > 0) northStarKey = "offline";
  else if (unhealthy > 0) northStarKey = "offline";
  else if (summary.outdatedAgents > 0) northStarKey = "agents";

  const items: readonly StatItem[] = [
    {
      key: "total",
      label: "Total",
      value: String(summary.total),
      hint: summary.total === 1 ? "server" : "servers",
      tone: "default",
    },
    {
      key: "online",
      label: "Online",
      value: summary.total === 0 ? "0" : `${summary.online}/${summary.total}`,
      hint: "reachable",
      tone: onlineTone,
    },
    {
      key: "offline",
      label: "Issues",
      value: String(unhealthy),
      hint:
        summary.error > 0
          ? `${summary.error} error${summary.error === 1 ? "" : "s"}`
          : summary.offline > 0
            ? `${summary.offline} offline`
            : summary.pending > 0
              ? `${summary.pending} pending`
              : "none",
      tone: offlineTone,
    },
    {
      key: "agents",
      label: "Agents",
      value: summary.latestAgentVersion
        ? `v${summary.latestAgentVersion}`
        : "—",
      hint:
        summary.outdatedAgents > 0
          ? `${summary.outdatedAgents} outdated`
          : "up to date",
      tone: agentsTone,
    },
  ];

  return items.map((item) =>
    item.key === northStarKey && item.tone !== "default"
      ? { ...item, isNorthStar: true }
      : item,
  );
}

function ServersNbaBanner({
  action,
}: {
  readonly action: ServersNextBestAction;
}) {
  const isDestructive = action.severity === "destructive";
  return (
    <Alert
      variant={isDestructive ? "destructive" : "default"}
      className={cn(
        "mb-5 flex items-start gap-3",
        !isDestructive &&
          "border-yellow-500/40 bg-yellow-500/5 text-yellow-700 dark:border-yellow-500/30 dark:text-yellow-300",
      )}
    >
      <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div className="flex-1 space-y-1">
        <AlertTitle className="text-sm font-semibold">
          {isDestructive ? "Action required" : "Needs your attention"}
        </AlertTitle>
        <AlertDescription className="text-xs sm:text-sm">
          {action.message}
        </AlertDescription>
      </div>
      <Link
        to={action.ctaHref}
        className="inline-flex shrink-0 items-center gap-1 self-center text-xs font-medium underline-offset-4 hover:underline"
      >
        {action.ctaLabel}
        <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
      </Link>
    </Alert>
  );
}

export function ServersHero({ servers, isLoading }: ServersHeroProps) {
  const queryClient = useQueryClient();

  const fleet = servers ?? [];
  const summary = summarizeServers(fleet);
  const subtitle = isLoading
    ? "Loading fleet status…"
    : buildServersSubtitle(summary);
  const nba = isLoading ? null : buildServersNextBestAction(summary);
  const stats = buildStats(summary);

  const handleRefresh = () => {
    void queryClient.invalidateQueries({ queryKey: SERVERS_QUERY_KEY });
  };

  return (
    <section
      aria-labelledby="servers-hero-title"
      className="relative overflow-hidden rounded-xl border border-border/60 bg-card"
    >
      <HeroBackground
        glowClassName="h-[280px] w-[280px] -right-20 -top-20 bg-emerald-400/5 dark:bg-emerald-500/5"
        dotOpacityClassName="opacity-40 dark:opacity-20"
      />

      <div className="relative px-4 py-5 sm:px-6 sm:py-6 lg:px-7 lg:py-7">
        {nba && <ServersNbaBanner action={nba} />}

        <div className="mb-3 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
          <Link
            to={ROUTES.HOME}
            className="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 transition-colors hover:bg-muted/60 hover:text-foreground"
          >
            <ArrowLeft className="h-3 w-3" aria-hidden="true" />
            <span>Dashboard</span>
          </Link>
          <span aria-hidden="true" className="text-border">
            /
          </span>
          <span className="inline-flex items-center gap-1 text-foreground">
            <Terminal className="h-3 w-3" aria-hidden="true" />
            <span>Servers</span>
          </span>
        </div>

        <h1
          id="servers-hero-title"
          className="text-2xl font-semibold tracking-tight text-foreground opacity-0 animate-fade-in-up [animation-delay:0.1s] sm:text-3xl"
        >
          <span className="inline-flex items-center gap-2">
            <ServerIcon
              className="h-6 w-6 text-muted-foreground"
              aria-hidden="true"
            />
            Remote Servers
          </span>
        </h1>
        <p className="mt-1.5 max-w-2xl text-sm text-muted-foreground opacity-0 animate-fade-in-up [animation-delay:0.2s]">
          {subtitle}
        </p>

        <dl className="mt-5 grid grid-cols-2 gap-x-6 gap-y-3 border-t border-border/50 pt-4 opacity-0 animate-fade-in-up [animation-delay:0.3s] sm:grid-cols-4">
          {stats.map((item) => (
            <div key={item.key} className="min-w-0">
              <dt className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {item.isNorthStar && (
                  <span
                    className={cn(
                      "h-1.5 w-1.5 rounded-full",
                      TONE_DOT[item.tone],
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
                    TONE_TEXT[item.tone],
                  )}
                >
                  <span className="text-lg font-semibold tracking-tight sm:text-xl">
                    {item.value}
                  </span>
                  <span className="truncate text-[11px] text-muted-foreground">
                    {item.hint}
                  </span>
                </dd>
              )}
            </div>
          ))}
        </dl>

        <div className="mt-5 flex flex-wrap items-center gap-1.5 border-t border-border/50 pt-4 opacity-0 animate-fade-in-up [animation-delay:0.4s]">
          <AddServerDialog
            trigger={
              <Button size="sm" className="h-9">
                <Plus className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
                Add Server
              </Button>
            }
          />
          <Button asChild variant="outline" size="sm" className="h-9">
            <Link to={ROUTES.HELPER_SERVER_SETUP}>
              <HelpCircle className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              Setup guide
            </Link>
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-9"
            onClick={handleRefresh}
            disabled={isLoading}
          >
            <RefreshCw
              className={cn("mr-1.5 h-3.5 w-3.5", isLoading && "animate-spin")}
              aria-hidden="true"
            />
            Refresh
          </Button>
        </div>
      </div>
    </section>
  );
}
