import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDate(date: string | Date): string {
  const d = new Date(date);
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatRelativeTime(date: string | Date): string {
  const d = new Date(date);
  const now = new Date();
  const diff = now.getTime() - d.getTime();

  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days}d ago`;
  if (hours > 0) return `${hours}h ago`;
  if (minutes > 0) return `${minutes}m ago`;
  return "just now";
}

export function truncateCommitSha(sha: string): string {
  return sha.slice(0, 7);
}

export function formatRepositoryUrl(url: string): string {
  return url.replace("https://github.com/", "");
}

export function sanitizeAppName(name: string): string {
  return name.toLowerCase().replaceAll(/[^a-z0-9-]/g, "-");
}

export function isValidGitHubUrl(url: string): boolean {
  return url.includes("github.com") || url.includes("git@github.com");
}

export function filterRepositories<
  T extends { name: string; fullName: string; description?: string },
>(repositories: readonly T[], query: string): T[] {
  if (!query) return [...repositories];

  const normalizedQuery = query.toLowerCase();
  return repositories.filter(
    (repo) =>
      repo.name.toLowerCase().includes(normalizedQuery) ||
      repo.fullName.toLowerCase().includes(normalizedQuery) ||
      repo.description?.toLowerCase().includes(normalizedQuery),
  );
}

export function formatDuration(milliseconds: number): string {
  const seconds = Math.floor(milliseconds / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    const remainingMinutes = minutes % 60;
    return remainingMinutes > 0
      ? `${hours}h ${remainingMinutes}m`
      : `${hours}h`;
  }
  if (minutes > 0) {
    const remainingSeconds = seconds % 60;
    return remainingSeconds > 0
      ? `${minutes}m ${remainingSeconds}s`
      : `${minutes}m`;
  }
  return `${seconds}s`;
}
