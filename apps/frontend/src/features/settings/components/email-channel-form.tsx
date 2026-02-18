import { Input } from "@/components/ui/input";

interface EmailChannelFormProps {
  readonly config: Record<string, string>;
  readonly onChange: (config: Record<string, string>) => void;
}

export function EmailChannelForm({ config, onChange }: EmailChannelFormProps) {
  return (
    <div className="space-y-3">
      <div className="space-y-2">
        <label htmlFor="smtpHost" className="text-sm font-medium leading-none">
          SMTP Host
        </label>
        <Input
          id="smtpHost"
          placeholder="smtp.gmail.com"
          value={config.smtpHost ?? ""}
          onChange={(e) => onChange({ ...config, smtpHost: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <label htmlFor="smtpPort" className="text-sm font-medium leading-none">
          SMTP Port
        </label>
        <Input
          id="smtpPort"
          type="number"
          placeholder="587"
          value={config.smtpPort ?? "587"}
          onChange={(e) => onChange({ ...config, smtpPort: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <label htmlFor="from" className="text-sm font-medium leading-none">
          From
        </label>
        <Input
          id="from"
          type="email"
          placeholder="sender@example.com"
          value={config.from ?? ""}
          onChange={(e) => onChange({ ...config, from: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <label htmlFor="to" className="text-sm font-medium leading-none">
          To
        </label>
        <Input
          id="to"
          type="email"
          placeholder="recipient@example.com"
          value={config.to ?? ""}
          onChange={(e) => onChange({ ...config, to: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <label htmlFor="username" className="text-sm font-medium leading-none">
          Username (optional)
        </label>
        <Input
          id="username"
          placeholder="SMTP username"
          value={config.username ?? ""}
          onChange={(e) => onChange({ ...config, username: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <label htmlFor="password" className="text-sm font-medium leading-none">
          Password (optional)
        </label>
        <Input
          id="password"
          type="password"
          placeholder="SMTP password"
          value={config.password ?? ""}
          onChange={(e) => onChange({ ...config, password: e.target.value })}
        />
      </div>
    </div>
  );
}
