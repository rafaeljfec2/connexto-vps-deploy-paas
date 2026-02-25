import type { CustomDomain } from "@/types";

export function buildDomainUrl(domain: CustomDomain): string {
  const rawPath = domain.pathPrefix?.trim() ?? "";
  let path = "";
  if (rawPath !== "") {
    path = rawPath.startsWith("/") ? rawPath : `/${rawPath}`;
  }
  return `https://${domain.domain}${path}`;
}

export function getOpenAppUrl(
  customDomains: readonly CustomDomain[],
  fallbackUrl: string | null,
): string | null {
  const rootDomain = customDomains.find(
    (domain) => domain.pathPrefix?.trim() === "",
  );
  if (rootDomain) {
    return buildDomainUrl(rootDomain);
  }
  const firstDomain = customDomains[0];
  if (firstDomain) {
    return buildDomainUrl(firstDomain);
  }
  return fallbackUrl;
}
