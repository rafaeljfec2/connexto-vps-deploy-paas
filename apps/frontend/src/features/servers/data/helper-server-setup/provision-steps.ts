export interface ProvisionStep {
  readonly id: string;
  readonly label: string;
}

export const PROVISION_STEPS: ReadonlyArray<ProvisionStep> = [
  { id: "ssh_connect", label: "Connecting via SSH" },
  { id: "remote_env", label: "Detecting environment (home dir, UID)" },
  {
    id: "docker_check / docker_install",
    label: "Checking / installing Docker",
  },
  { id: "docker_start", label: "Checking / starting Docker daemon" },
  { id: "docker_network", label: "Creating Docker network" },
  {
    id: "traefik_check / traefik_install",
    label: "Checking / installing Traefik",
  },
  { id: "sftp_client", label: "Connecting SFTP" },
  { id: "install_dir", label: "Creating directories" },
  { id: "agent_certs", label: "Generating and installing mTLS certificates" },
  { id: "agent_binary", label: "Copying agent binary" },
  { id: "systemd_unit", label: "Configuring systemd service" },
  { id: "start_agent", label: "Starting agent" },
];
