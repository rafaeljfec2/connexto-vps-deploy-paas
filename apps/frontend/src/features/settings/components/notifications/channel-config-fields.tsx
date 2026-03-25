import { DiscordChannelForm } from "@/features/settings/components/discord-channel-form";
import { EmailChannelForm } from "@/features/settings/components/email-channel-form";
import { SlackChannelForm } from "@/features/settings/components/slack-channel-form";
import type { NotificationChannelType } from "@/types";

interface ChannelConfigFieldsProps {
  readonly type: NotificationChannelType;
  readonly config: Record<string, string>;
  readonly onChange: (config: Record<string, string>) => void;
}

export function ChannelConfigFields({
  type,
  config,
  onChange,
}: ChannelConfigFieldsProps) {
  if (type === "slack") {
    return <SlackChannelForm config={config} onChange={onChange} />;
  }
  if (type === "discord") {
    return <DiscordChannelForm config={config} onChange={onChange} />;
  }
  if (type === "email") {
    return <EmailChannelForm config={config} onChange={onChange} />;
  }
  return null;
}
