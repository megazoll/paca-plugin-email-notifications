import type { PluginApiClient } from "@paca-ai/plugin-sdk-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PLUGIN_ID } from "./constants";
import type {
  EmailLog,
  SMTPSettings,
  UpdateSettingsInput,
  UserEmailOverride,
} from "./types";

async function unwrapData<T>(promise: Promise<any>): Promise<T> {
  const res = await promise;
  if (res && typeof res === "object") {
    if ("error" in res && res.error && !("data" in res)) {
      throw new Error(typeof res.error === "string" ? res.error : JSON.stringify(res.error));
    }
    if ("data" in res && res.data !== undefined) {
      return res.data as T;
    }
  }
  return res as T;
}

// ── Project Scope Hooks ──────────────────────────────────────────────────────

export function useProjectEmailSettings(api: PluginApiClient, projectId?: string) {
  const pid = projectId || api.projectId;
  return useQuery({
    queryKey: [PLUGIN_ID, "settings", "project", pid],
    queryFn: () =>
      unwrapData<SMTPSettings>(
        api.get(`/projects/${pid}/email-notifications/settings`),
      ),
    enabled: Boolean(pid),
  });
}

export function useUpdateProjectEmailSettings(api: PluginApiClient, projectId?: string) {
  const queryClient = useQueryClient();
  const pid = projectId || api.projectId;
  return useMutation({
    mutationFn: (input: UpdateSettingsInput) =>
      unwrapData<SMTPSettings>(
        api.patch(`/projects/${pid}/email-notifications/settings`, input),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [PLUGIN_ID, "settings", "project", pid],
      });
    },
  });
}

export function useProjectEmailLogs(api: PluginApiClient, projectId?: string) {
  const pid = projectId || api.projectId;
  return useQuery({
    queryKey: [PLUGIN_ID, "logs", "project", pid],
    queryFn: () =>
      unwrapData<EmailLog[]>(
        api.get(`/projects/${pid}/email-notifications/logs`),
      ),
    enabled: Boolean(pid),
    refetchInterval: 10000,
  });
}

export function useSendProjectTestEmail(api: PluginApiClient, projectId?: string) {
  const queryClient = useQueryClient();
  const pid = projectId || api.projectId;
  return useMutation({
    mutationFn: (toEmail: string) =>
      unwrapData<{ sent: boolean; recipient: string }>(
        api.post(`/projects/${pid}/email-notifications/test`, { to_email: toEmail }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [PLUGIN_ID, "logs", "project", pid],
      });
    },
  });
}

// ── Admin Scope Hooks ────────────────────────────────────────────────────────

export function useAdminEmailSettings(api: PluginApiClient) {
  return useQuery({
    queryKey: [PLUGIN_ID, "settings", "admin"],
    queryFn: () =>
      unwrapData<SMTPSettings>(
        api.get("/email-notifications/admin-settings"),
      ),
  });
}

export function useUpdateAdminEmailSettings(api: PluginApiClient) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateSettingsInput) =>
      unwrapData<SMTPSettings>(
        api.patch("/email-notifications/admin-settings", input),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [PLUGIN_ID, "settings", "admin"],
      });
    },
  });
}

export function useAdminEmailLogs(api: PluginApiClient) {
  return useQuery({
    queryKey: [PLUGIN_ID, "logs", "admin"],
    queryFn: () =>
      unwrapData<EmailLog[]>(
        api.get("/email-notifications/admin-logs"),
      ),
    refetchInterval: 10000,
  });
}

export function useSendAdminTestEmail(api: PluginApiClient) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (toEmail: string) =>
      unwrapData<{ sent: boolean; recipient: string }>(
        api.post("/email-notifications/admin-test", { to_email: toEmail }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [PLUGIN_ID, "logs", "admin"],
      });
    },
  });
}

export function useAdminEmailOverrides(api: PluginApiClient) {
  return useQuery({
    queryKey: [PLUGIN_ID, "overrides"],
    queryFn: () =>
      unwrapData<UserEmailOverride[]>(
        api.get("/email-notifications/admin-overrides"),
      ),
  });
}

export function useSaveAdminEmailOverride(api: PluginApiClient) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, email }: { userId: string; email: string }) =>
      unwrapData<{ saved: boolean; user_id: string; email: string }>(
        api.post("/email-notifications/admin-overrides", { user_id: userId, email }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [PLUGIN_ID, "overrides"],
      });
    },
  });
}

export function useDeleteAdminEmailOverride(api: PluginApiClient) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      unwrapData<{ deleted: boolean; user_id: string }>(
        api.delete(`/email-notifications/admin-overrides/${userId}`),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [PLUGIN_ID, "overrides"],
      });
    },
  });
}
