import type {
  CreateNotificationChannelInput,
  CreateNotificationRuleInput,
  NotificationChannel,
  NotificationRule,
} from "@/types";
import { API_BASE, fetchApi, fetchApiDelete, fetchApiList } from "./client";

export const notificationsApi = {
  channels: {
    list: (appId?: string): Promise<readonly NotificationChannel[]> => {
      const url = appId
        ? `${API_BASE}/notifications/channels?appId=${appId}`
        : `${API_BASE}/notifications/channels`;
      return fetchApiList<NotificationChannel>(url);
    },

    get: (id: string): Promise<NotificationChannel> =>
      fetchApi<NotificationChannel>(`${API_BASE}/notifications/channels/${id}`),

    create: (
      input: CreateNotificationChannelInput,
    ): Promise<NotificationChannel> =>
      fetchApi<NotificationChannel>(`${API_BASE}/notifications/channels`, {
        method: "POST",
        body: JSON.stringify(input),
      }),

    update: (
      id: string,
      input: { name?: string; config?: Record<string, unknown> },
    ): Promise<NotificationChannel> =>
      fetchApi<NotificationChannel>(
        `${API_BASE}/notifications/channels/${id}`,
        { method: "PUT", body: JSON.stringify(input) },
      ),

    delete: (id: string): Promise<void> =>
      fetchApiDelete(`${API_BASE}/notifications/channels/${id}`),

    rules: (channelId: string): Promise<readonly NotificationRule[]> =>
      fetchApiList<NotificationRule>(
        `${API_BASE}/notifications/channels/${channelId}/rules`,
      ),
  },

  rules: {
    list: (channelId?: string): Promise<readonly NotificationRule[]> => {
      const url = channelId
        ? `${API_BASE}/notifications/rules?channelId=${channelId}`
        : `${API_BASE}/notifications/rules`;
      return fetchApiList<NotificationRule>(url);
    },

    get: (id: string): Promise<NotificationRule> =>
      fetchApi<NotificationRule>(`${API_BASE}/notifications/rules/${id}`),

    create: (input: CreateNotificationRuleInput): Promise<NotificationRule> =>
      fetchApi<NotificationRule>(`${API_BASE}/notifications/rules`, {
        method: "POST",
        body: JSON.stringify(input),
      }),

    update: (
      id: string,
      input: { eventType?: string; enabled?: boolean },
    ): Promise<NotificationRule> =>
      fetchApi<NotificationRule>(`${API_BASE}/notifications/rules/${id}`, {
        method: "PUT",
        body: JSON.stringify(input),
      }),

    delete: (id: string): Promise<void> =>
      fetchApiDelete(`${API_BASE}/notifications/rules/${id}`),
  },
};
