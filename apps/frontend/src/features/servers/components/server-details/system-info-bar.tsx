import { formatBytes } from "@/lib/format";
import type { ServerStats } from "@/types";

function formatOsInfo(
  os: string | undefined,
  osVersion: string | undefined,
): string {
  if (os != null && osVersion != null) return `${os} ${osVersion}`;
  return os ?? osVersion ?? "—";
}

interface SystemInfoBarProps {
  readonly stats: ServerStats;
}

export function SystemInfoBar({ stats }: SystemInfoBarProps) {
  const { systemInfo } = stats;
  const items = [
    { label: "Host", value: systemInfo.hostname },
    { label: "OS", value: formatOsInfo(systemInfo.os, systemInfo.os_version) },
    { label: "Arch", value: systemInfo.architecture },
    { label: "Cores", value: String(systemInfo.cpu_cores ?? "—") },
    { label: "Kernel", value: systemInfo.kernel_version },
    { label: "RAM", value: formatBytes(systemInfo.memory_total_bytes ?? 0) },
    { label: "Disk", value: formatBytes(systemInfo.disk_total_bytes ?? 0) },
  ];

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
      {items.map((item) => (
        <span key={item.label}>
          <span className="font-medium text-foreground/70">{item.label}:</span>{" "}
          {item.value ?? "—"}
        </span>
      ))}
    </div>
  );
}
