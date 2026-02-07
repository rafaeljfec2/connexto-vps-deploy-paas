import { type FormEvent, useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { ExclamationTriangleIcon } from "@radix-ui/react-icons";
import { ROUTES } from "@/constants/routes";
import { useAuth } from "@/contexts/auth-context";
import { Check, Loader2, Rocket } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api } from "@/services/api";
import { ApiError } from "@/types";

type AuthErrorCode =
  | "no_code"
  | "invalid_state"
  | "token_exchange_failed"
  | "user_fetch_failed"
  | "encryption_failed"
  | "user_creation_failed"
  | "user_update_failed"
  | "session_error"
  | "database_error"
  | "access_denied"
  | "github_already_linked"
  | "link_failed"
  | "not_authenticated";

const ERROR_MESSAGES: Record<AuthErrorCode, string> = {
  no_code: "GitHub did not return an authorization code.",
  invalid_state: "The security state parameter is invalid. Please try again.",
  token_exchange_failed:
    "Failed to exchange the authorization code for a token.",
  user_fetch_failed: "Failed to fetch your GitHub profile.",
  encryption_failed: "An internal error occurred. Please try again.",
  user_creation_failed: "Failed to create your account. Please try again.",
  user_update_failed: "Failed to update your account. Please try again.",
  session_error: "Failed to create a session. Please try again.",
  database_error: "A database error occurred. Please try again.",
  access_denied: "You denied access to your GitHub account.",
  github_already_linked:
    "This GitHub account is already linked to another user.",
  link_failed: "Failed to link your GitHub account. Please try again.",
  not_authenticated: "You must be logged in to link a GitHub account.",
};

function isAuthErrorCode(code: string): code is AuthErrorCode {
  return code in ERROR_MESSAGES;
}

function getErrorMessage(error: string | null): string | null {
  if (!error) return null;
  return isAuthErrorCode(error)
    ? ERROR_MESSAGES[error]
    : `An unknown error occurred: ${error}`;
}

function GitHubLogo({ className }: { readonly className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden
    >
      <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
    </svg>
  );
}

function DockerLogo({ className }: { readonly className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden
    >
      <path d="M13.983 10.779h3.782V7.002h-3.782v3.777zm-5.393 0h3.782V7.002H8.59v3.777zm5.393 5.395h3.782v-3.777h-3.782v3.777zm-5.393 0h3.782v-3.777H8.59v3.777zm5.393-10.79h3.782V1.607h-3.782v3.777zM8.59 5.395h3.782V1.607H8.59v3.788zM2.197 10.779h3.782V7.002H2.197v3.777zm0 5.395h3.782v-3.777H2.197v3.777zm15.568 0h3.782v-3.777h-3.782v3.777zM2.197 5.395h3.782V1.607H2.197v3.788z" />
    </svg>
  );
}

function CloudflareLogo({ className }: { readonly className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden
    >
      <path d="M15.8 12.5l-3.5-6c-.2-.3-.5-.5-.8-.5s-.6.2-.8.5l-4 6.9c-.2.3-.2.7 0 1 .2.3.5.5.8.5h2.5v5c0 .3.2.5.5.5h2c.3 0 .5-.2.5-.5v-5h1.5c.3 0 .6-.2.8-.5.1-.3.1-.7-.2-1z" />
    </svg>
  );
}

const FEATURES = [
  "Git-based automatic deployments",
  "Automatic SSL certificates",
  "Real-time logs and monitoring",
  "Custom domains with Cloudflare",
  "One-click rollbacks",
  "Container console access",
] as const;

function EmailLoginForm({
  errorMessage,
  onSuccess,
}: {
  readonly errorMessage: string | null;
  readonly onSuccess: () => void;
}) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      await api.auth.login({ email, password });
      onSuccess();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred. Please try again.");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const displayError = error ?? errorMessage;

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {displayError && (
        <Alert variant="destructive" role="alert">
          <ExclamationTriangleIcon className="h-4 w-4" aria-hidden="true" />
          <AlertTitle>Authentication Error</AlertTitle>
          <AlertDescription>{displayError}</AlertDescription>
        </Alert>
      )}

      <div className="space-y-2">
        <Label htmlFor="email">Email</Label>
        <Input
          id="email"
          type="email"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoComplete="email"
          disabled={isSubmitting}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="password">Password</Label>
        <Input
          id="password"
          type="password"
          placeholder="••••••••"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={8}
          autoComplete="current-password"
          disabled={isSubmitting}
        />
      </div>

      <Button
        type="submit"
        className="h-11 w-full"
        size="lg"
        disabled={isSubmitting}
      >
        {isSubmitting ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden />
            Signing in...
          </>
        ) : (
          "Sign in"
        )}
      </Button>
    </form>
  );
}

export function LoginPage() {
  const { isAuthenticated, isLoading, login, refresh } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const errorMessage = getErrorMessage(searchParams.get("error"));

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      navigate(ROUTES.HOME, { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate]);

  const handleLoginSuccess = () => {
    refresh();
  };

  if (isLoading) {
    return (
      <div className="flex h-full min-h-0 items-center justify-center px-4">
        <div
          className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"
          aria-hidden="true"
        />
        <output className="sr-only" aria-live="polite">
          Checking authentication status
        </output>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden md:flex-row">
      <aside className="relative flex min-h-0 flex-col justify-between overflow-hidden bg-slate-100 bg-gradient-to-br from-slate-50 to-slate-200 px-6 py-8 dark:from-slate-900 dark:to-slate-800 dark:bg-slate-900 md:w-[40%] md:py-10 lg:w-1/2 lg:px-12 lg:py-12">
        <div
          className="pointer-events-none absolute inset-0 opacity-100 dark:opacity-0"
          style={{
            backgroundImage:
              "radial-gradient(circle, rgba(0,0,0,0.06) 1px, transparent 1px)",
            backgroundSize: "24px 24px",
          }}
          aria-hidden
        />
        <div
          className="pointer-events-none absolute inset-0 opacity-0 dark:opacity-100"
          style={{
            backgroundImage:
              "radial-gradient(circle, rgba(255,255,255,0.08) 1px, transparent 1px)",
            backgroundSize: "24px 24px",
          }}
          aria-hidden
        />
        <div
          className="pointer-events-none absolute -right-24 -top-24 h-96 w-96 rounded-full bg-emerald-400/15 blur-3xl dark:bg-emerald-500/20"
          aria-hidden
        />
        <div className="relative min-h-0 shrink-0">
          <div className="flex flex-wrap items-center gap-2 text-slate-900 dark:text-white">
            <Rocket
              className="h-8 w-8 shrink-0 text-slate-900 dark:text-white"
              aria-hidden
            />
            <span className="text-xl font-semibold tracking-tight">
              FlowDeploy
            </span>
            <span className="rounded-md border border-slate-300 bg-slate-200/80 px-2 py-0.5 text-xs font-medium text-slate-700 dark:border-slate-600 dark:bg-slate-800/80 dark:text-slate-300">
              Self-hosted
            </span>
          </div>
          <h1 className="mt-8 text-2xl font-bold text-slate-900 sm:text-3xl dark:text-white md:mt-8 lg:mt-10 lg:text-4xl">
            Deploy your applications with confidence
          </h1>
          <p className="mt-2 text-slate-600 text-sm dark:text-slate-400 lg:text-base">
            The self-hosted PaaS that gives you full control
          </p>
          <ul className="mt-6 hidden space-y-3 md:mt-8 md:block lg:mt-8 lg:space-y-4">
            {FEATURES.map((feature) => (
              <li
                key={feature}
                className="flex items-center gap-3 text-slate-700 text-sm transition-colors hover:text-slate-900 dark:text-slate-300 dark:hover:text-slate-200 lg:text-base"
              >
                <Check
                  className="h-5 w-5 shrink-0 text-emerald-600 dark:text-emerald-500"
                  aria-hidden
                />
                <span>{feature}</span>
              </li>
            ))}
          </ul>
        </div>
        <div className="relative mt-6 flex shrink-0 flex-wrap items-center justify-center gap-4 border-t border-slate-300 pt-6 dark:border-slate-700/80 md:justify-start">
          <span className="text-slate-600 text-xs dark:text-slate-500">
            Powered by
          </span>
          <div className="flex items-center gap-5 text-slate-500 dark:text-slate-400">
            <DockerLogo className="h-5 w-5" />
            <GitHubLogo className="h-5 w-5" />
            <CloudflareLogo className="h-5 w-5" />
          </div>
        </div>
      </aside>

      <main className="flex min-h-0 flex-1 flex-col items-center justify-center bg-slate-50 px-6 py-8 dark:bg-background md:w-[60%] md:py-10 lg:w-1/2 lg:px-12 lg:py-12">
        <Card className="w-full max-w-md border-slate-200 shadow-lg dark:border-border/80 dark:shadow-md">
          <CardHeader className="space-y-1.5 pb-4">
            <div className="flex justify-center md:justify-start">
              <Rocket className="h-10 w-10 shrink-0 text-primary" aria-hidden />
            </div>
            <CardTitle className="text-2xl">Sign in to FlowDeploy</CardTitle>
            <CardDescription>
              Welcome back! Please sign in to continue.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <EmailLoginForm
              errorMessage={errorMessage}
              onSuccess={handleLoginSuccess}
            />

            <div className="relative flex items-center gap-3">
              <span
                className="flex-1 border-t border-slate-200 dark:border-border"
                aria-hidden
              />
              <span className="text-muted-foreground text-xs">or</span>
              <span
                className="flex-1 border-t border-slate-200 dark:border-border"
                aria-hidden
              />
            </div>

            <Button
              onClick={login}
              variant="outline"
              className="h-11 w-full"
              size="lg"
              aria-label="Sign in with GitHub"
            >
              <GitHubLogo className="mr-2 h-5 w-5 shrink-0" />
              Continue with GitHub
            </Button>

            <p className="text-center text-sm text-muted-foreground">
              Don&apos;t have an account?{" "}
              <Link
                to={ROUTES.REGISTER}
                className="font-medium text-primary underline-offset-4 hover:underline"
              >
                Sign up
              </Link>
            </p>
          </CardContent>
          <CardFooter className="flex justify-center border-t border-slate-200 pt-6 dark:border-border/50">
            <p className="text-center text-muted-foreground text-xs">
              By signing in, you agree to our{" "}
              <Link
                to={ROUTES.TERMS}
                className="text-primary underline hover:opacity-90 dark:hover:text-foreground"
              >
                Terms of Service
              </Link>
            </p>
          </CardFooter>
        </Card>
      </main>
    </div>
  );
}
