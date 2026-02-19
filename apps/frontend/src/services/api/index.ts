import {
  appsApi,
  containerApi,
  deploymentsApi,
  domainsApi,
  envVarsApi,
  webhooksApi,
} from "./apps";
import { authApi, githubApi } from "./auth";
import { containersApi, imagesApi, networksApi, volumesApi } from "./docker";
import {
  certificatesApi,
  cloudflareApi,
  migrationApi,
  templatesApi,
} from "./infrastructure";
import { notificationsApi } from "./notifications";
import { serversApi } from "./servers";

export type {
  GitHubInstallation,
  GitHubRepository,
  LoginInput,
  RegisterInput,
  ReposResponse,
} from "./auth";
export type {
  DockerImageInfo,
  DockerNetworkInfo,
  DockerVolumeInfo,
} from "./docker";

export { API_BASE, API_URL } from "./client";

export const api = {
  auth: authApi,
  github: githubApi,
  apps: appsApi,
  deployments: deploymentsApi,
  container: containerApi,
  webhooks: webhooksApi,
  envVars: envVarsApi,
  domains: domainsApi,
  containers: containersApi,
  images: imagesApi,
  networks: networksApi,
  volumes: volumesApi,
  servers: serversApi,
  notifications: notificationsApi,
  cloudflare: cloudflareApi,
  certificates: certificatesApi,
  migration: migrationApi,
  templates: templatesApi,
} as const;
