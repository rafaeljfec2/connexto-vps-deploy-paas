import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
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
