import { type FormEvent, useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { ExclamationTriangleIcon } from "@radix-ui/react-icons";
import { ROUTES } from "@/constants/routes";
import { useAuth } from "@/contexts/auth-context";
import { Loader2, Rocket } from "lucide-react";
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
import { ThemeToggle } from "@/components/theme-toggle";
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
      <div className="flex h-dvh items-center justify-center">
        <div
          className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"
          aria-hidden="true"
        />
      </div>
    );
  }

  return (
    <div className="flex min-h-dvh flex-col bg-background">
      <header className="flex items-center justify-between px-4 py-4 sm:px-6">
        <Link
          to={ROUTES.LANDING}
          className="flex items-center gap-2 text-lg font-semibold"
          aria-label="flowDeploy - Home"
        >
          <Rocket className="h-6 w-6" aria-hidden="true" />
          <span>flowDeploy</span>
        </Link>
        <ThemeToggle />
      </header>

      <main className="flex flex-1 items-center justify-center px-4 py-8">
        <Card className="w-full max-w-md shadow-lg">
          <CardHeader className="space-y-1.5 pb-4">
            <CardTitle className="text-2xl">Sign in to flowDeploy</CardTitle>
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
              <span className="flex-1 border-t border-border" aria-hidden />
              <span className="text-muted-foreground text-xs">or</span>
              <span className="flex-1 border-t border-border" aria-hidden />
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
          <CardFooter className="flex justify-center border-t pt-6">
            <p className="text-center text-muted-foreground text-xs">
              By signing in, you agree to our{" "}
              <Link
                to={ROUTES.TERMS}
                className="text-primary underline hover:opacity-90"
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
