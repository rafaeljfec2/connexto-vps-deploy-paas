import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ExclamationTriangleIcon, GitHubLogoIcon } from "@radix-ui/react-icons";
import { useAuth } from "@/contexts/auth-context";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const ERROR_MESSAGES: Record<string, string> = {
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
};

export function LoginPage() {
  const { isAuthenticated, isLoading, login } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const error = searchParams.get("error");
  const errorMessage = error
    ? (ERROR_MESSAGES[error] ?? `An unknown error occurred: ${error}`)
    : null;

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      navigate("/", { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-background">
      <Card className="w-full max-w-md mx-4">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold">FlowDeploy</CardTitle>
          <CardDescription>Sign in to deploy your applications</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {errorMessage && (
            <Alert variant="destructive">
              <ExclamationTriangleIcon className="h-4 w-4" />
              <AlertTitle>Authentication Error</AlertTitle>
              <AlertDescription>{errorMessage}</AlertDescription>
            </Alert>
          )}

          <Button onClick={login} className="w-full" size="lg">
            <GitHubLogoIcon className="mr-2 h-5 w-5" />
            Sign in with GitHub
          </Button>

          <p className="text-xs text-center text-muted-foreground">
            By signing in, you agree to our terms of service and privacy policy.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
