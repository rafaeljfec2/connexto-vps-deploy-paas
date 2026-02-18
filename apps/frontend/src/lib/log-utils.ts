export interface ContainerLogLine {
  readonly lineNumber: number;
  readonly timestamp: string | null;
  readonly content: string;
  readonly type: "info" | "error" | "warning" | "default";
}

export type DeployLogType =
  | "info"
  | "success"
  | "error"
  | "warning"
  | "build"
  | "default";
export type DeployLogPrefix = "build" | "deploy" | null;

export interface DeployLogLine {
  readonly lineNumber: number;
  readonly timestamp: string | null;
  readonly prefix: DeployLogPrefix;
  readonly step: string | null;
  readonly content: string;
  readonly type: DeployLogType;
  readonly isEmpty: boolean;
}

const DOCKER_TIMESTAMP_REGEX =
  /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s*/;

const DEPLOY_TIMESTAMP_REGEX = /^\[(\d{2}:\d{2}:\d{2})\]\s*/;
const DEPLOY_PREFIX_REGEX = /^\[(build|deploy)\]\s*/i;
const DEPLOY_STEP_REGEX = /^(#\d+)\s+/;

function determineContainerLogType(
  content: string,
): "info" | "error" | "warning" | "default" {
  const lower = content.toLowerCase();

  if (
    lower.includes("error") ||
    lower.includes("failed") ||
    lower.includes("fatal") ||
    lower.includes("exception") ||
    lower.includes("panic")
  ) {
    return "error";
  }

  if (
    lower.includes("warning") ||
    lower.includes("warn") ||
    lower.includes("deprecated")
  ) {
    return "warning";
  }

  if (
    lower.includes("info") ||
    lower.includes("starting") ||
    lower.includes("listening") ||
    lower.includes("connected")
  ) {
    return "info";
  }

  return "default";
}

function determineDeployLogType(
  content: string,
  prefix: DeployLogPrefix,
): DeployLogType {
  const lower = content.toLowerCase();

  if (
    lower.includes("error") ||
    lower.includes("failed") ||
    lower.includes("fatal") ||
    lower.includes("exception") ||
    lower.includes("could not be found")
  ) {
    return "error";
  }

  if (
    lower.includes("success") ||
    lower.includes("completed") ||
    lower.includes("deployed") ||
    lower.includes("healthy") ||
    lower.includes("running") ||
    lower.includes("done") ||
    lower.includes("cached")
  ) {
    return "success";
  }

  if (
    lower.includes("warning") ||
    lower.includes("warn") ||
    lower.includes("deprecated") ||
    lower.includes("obsolete")
  ) {
    return "warning";
  }

  if (
    lower.includes("starting") ||
    lower.includes("syncing") ||
    lower.includes("fetching") ||
    lower.includes("building") ||
    lower.includes("checking") ||
    lower.includes("pulling") ||
    lower.includes("pushing") ||
    lower.includes("deploying") ||
    lower.includes("exporting") ||
    lower.includes("transferring") ||
    lower.includes("unpacking") ||
    lower.includes("naming")
  ) {
    return "info";
  }

  if (prefix === "build") {
    return "build";
  }

  return "default";
}

function utcTimeToLocal(utcTime: string): string {
  const [hours, minutes, seconds] = utcTime.split(":").map(Number);
  const now = new Date();
  const utcDate = new Date(
    Date.UTC(
      now.getUTCFullYear(),
      now.getUTCMonth(),
      now.getUTCDate(),
      hours,
      minutes,
      seconds,
    ),
  );
  return utcDate.toLocaleTimeString("pt-BR", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

export function parseContainerLogLine(
  line: string,
  index: number,
): ContainerLogLine {
  let remaining = line;
  let timestamp: string | null = null;

  const timestampMatch = DOCKER_TIMESTAMP_REGEX.exec(remaining);
  if (timestampMatch?.[1]) {
    const date = new Date(timestampMatch[1]);
    timestamp = date.toLocaleTimeString("pt-BR", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
    remaining = remaining.slice(timestampMatch[0].length);
  }

  const content = remaining.trim();
  const type = determineContainerLogType(content);

  return {
    lineNumber: index + 1,
    timestamp,
    content,
    type,
  };
}

export function parseDeployLogLine(line: string, index: number): DeployLogLine {
  let remaining = line;
  let timestamp: string | null = null;
  let prefix: DeployLogPrefix = null;
  let step: string | null = null;

  const timestampMatch = DEPLOY_TIMESTAMP_REGEX.exec(remaining);
  if (timestampMatch?.[1]) {
    timestamp = utcTimeToLocal(timestampMatch[1]);
    remaining = remaining.slice(timestampMatch[0].length);
  }

  const prefixMatch = DEPLOY_PREFIX_REGEX.exec(remaining);
  if (prefixMatch?.[1]) {
    prefix = prefixMatch[1].toLowerCase() as DeployLogPrefix;
    remaining = remaining.slice(prefixMatch[0].length);
  }

  const stepMatch = DEPLOY_STEP_REGEX.exec(remaining);
  if (stepMatch?.[1]) {
    step = stepMatch[1];
    remaining = remaining.slice(stepMatch[0].length);
  }

  const content = remaining.trim();
  const isEmpty = content === "" || content === "...";
  const type = determineDeployLogType(content, prefix);

  return {
    lineNumber: index + 1,
    timestamp,
    prefix,
    step,
    content,
    type,
    isEmpty,
  };
}
