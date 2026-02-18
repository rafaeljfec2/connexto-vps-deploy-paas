import { Input } from "@/components/ui/input";

interface SlackChannelFormProps {
  readonly config: Record<string, string>;
  readonly onChange: (config: Record<string, string>) => void;
}

export function SlackChannelForm({ config, onChange }: SlackChannelFormProps) {
  return (
    <div className="space-y-2">
      <label htmlFor="webhookUrl" className="text-sm font-medium leading-none">
        Webhook URL
      </label>
      <Input
        id="webhookUrl"
        type="url"
        placeholder="https://hooks.slack.com/services/..."
        value={config.webhookUrl ?? ""}
        onChange={(e) => onChange({ ...config, webhookUrl: e.target.value })}
      />
    </div>
  );
}
