import { Input } from "@/components/ui/input";

interface DiscordChannelFormProps {
  readonly config: Record<string, string>;
  readonly onChange: (config: Record<string, string>) => void;
}

export function DiscordChannelForm({
  config,
  onChange,
}: DiscordChannelFormProps) {
  return (
    <div className="space-y-2">
      <label htmlFor="webhookUrl" className="text-sm font-medium leading-none">
        Webhook URL
      </label>
      <Input
        id="webhookUrl"
        type="url"
        placeholder="https://discord.com/api/webhooks/..."
        value={config.webhookUrl ?? ""}
        onChange={(e) => onChange({ ...config, webhookUrl: e.target.value })}
      />
    </div>
  );
}
