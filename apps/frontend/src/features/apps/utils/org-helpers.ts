/**
 * Extracts the repository owner (organization or user) from a Git repository URL.
 * Supports HTTPS, HTTP, and SSH formats.
 */
export function extractRepoOwner(repositoryUrl: string): string | null {
  if (!repositoryUrl) return null;

  const trimmed = repositoryUrl.trim();
  if (trimmed === "") return null;

  // Handle SSH format: git@github.com:owner/repo.git
  const sshMatch = trimmed.match(/git@[^:]+:([^/]+)\/[^/]+(?:\.git)?\/?$/);
  if (sshMatch?.[1]) {
    return sshMatch[1];
  }

  // Handle HTTPS format: https://github.com/owner/repo.git
  try {
    const url = new URL(trimmed);
    const pathParts = url.pathname.split("/").filter(Boolean);
    if (pathParts.length >= 2) {
      return pathParts[0] ?? null;
    }
  } catch {
    // If not a full URL, attempt slash split fallback
    const cleaned = trimmed.replace(/\.git\/?$/, "");
    const parts = cleaned.split("/").filter(Boolean);
    if (parts.length >= 2) {
      const candidate = parts[parts.length - 2];
      if (candidate && !candidate.includes(":")) {
        return candidate;
      }
    }
  }

  return null;
}
