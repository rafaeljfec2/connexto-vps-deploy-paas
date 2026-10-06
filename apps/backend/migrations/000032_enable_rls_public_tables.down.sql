-- Reverse of 000032: disable RLS on public tables.
-- Does not restore anon/authenticated grants (those are Supabase-specific
-- defaults and must not be re-opened in production).

ALTER TABLE public.app_env_vars DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.apps DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.audit_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.cleanup_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.cloudflare_connections DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.custom_domains DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.deployments DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.github_installations DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.notification_channels DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.notification_rules DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.personal_access_tokens DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.pki_ca DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.schema_migrations DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.servers DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_installations DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.users DISABLE ROW LEVEL SECURITY;
ALTER TABLE public.webhook_payloads DISABLE ROW LEVEL SECURITY;
