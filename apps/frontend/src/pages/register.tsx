import { type FormEvent, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
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
import { api } from "@/services/api";
import { ApiError } from "@/types";

const MIN_PASSWORD_LENGTH = 8;

function validateForm(fields: {
  readonly name: string;
  readonly email: string;
  readonly password: string;
  readonly confirmPassword: string;
}): string | null {
  if (!fields.name.trim()) {
    return "Name is required.";
  }
  if (!fields.email.trim()) {
    return "Email is required.";
  }
  if (fields.password.length < MIN_PASSWORD_LENGTH) {
    return `Password must be at least ${MIN_PASSWORD_LENGTH} characters.`;
  }
  if (fields.password !== fields.confirmPassword) {
    return "Passwords do not match.";
  }
  return null;
}

export function RegisterPage() {
  const { isAuthenticated, isLoading, refresh } = useAuth();
  const navigate = useNavigate();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      navigate(ROUTES.HOME, { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);

    const validationError = validateForm({
      name,
      email,
      password,
      confirmPassword,
    });
    if (validationError) {
      setError(validationError);
      return;
    }

    setIsSubmitting(true);

    try {
      await api.auth.register({ email, password, name: name.trim() });
      refresh();
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

  if (isLoading) {
    return (
      <div className="flex min-h-dvh items-center justify-center px-4">
        <div
          className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"
          aria-hidden="true"
        />
        <output className="sr-only" aria-live="polite">
          Loading...
        </output>
      </div>
    );
  }

  return (
    <div className="flex min-h-dvh items-center justify-center bg-slate-50 px-4 py-8 dark:bg-background">
      <Card className="w-full max-w-md border-slate-200 shadow-lg dark:border-border/80 dark:shadow-md">
        <CardHeader className="space-y-1.5 pb-4">
          <div className="flex justify-center">
            <Rocket className="h-10 w-10 shrink-0 text-primary" aria-hidden />
          </div>
          <CardTitle className="text-center text-2xl">
            Create your account
          </CardTitle>
          <CardDescription className="text-center">
            Get started with FlowDeploy
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <Alert variant="destructive" role="alert">
                <ExclamationTriangleIcon
                  className="h-4 w-4"
                  aria-hidden="true"
                />
                <AlertTitle>Registration Error</AlertTitle>
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <Label htmlFor="register-name">Name</Label>
              <Input
                id="register-name"
                type="text"
                placeholder="John Doe"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                autoComplete="name"
                disabled={isSubmitting}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="register-email">Email</Label>
              <Input
                id="register-email"
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
              <Label htmlFor="register-password">Password</Label>
              <Input
                id="register-password"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={MIN_PASSWORD_LENGTH}
                autoComplete="new-password"
                disabled={isSubmitting}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="register-confirm-password">
                Confirm Password
              </Label>
              <Input
                id="register-confirm-password"
                type="password"
                placeholder="••••••••"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                minLength={MIN_PASSWORD_LENGTH}
                autoComplete="new-password"
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
                  Creating account...
                </>
              ) : (
                "Create account"
              )}
            </Button>
          </form>

          <p className="mt-4 text-center text-sm text-muted-foreground">
            Already have an account?{" "}
            <Link
              to={ROUTES.LOGIN}
              className="font-medium text-primary underline-offset-4 hover:underline"
            >
              Sign in
            </Link>
          </p>
        </CardContent>
        <CardFooter className="flex justify-center border-t border-slate-200 pt-6 dark:border-border/50">
          <p className="text-center text-muted-foreground text-xs">
            By creating an account, you agree to our{" "}
            <Link
              to={ROUTES.TERMS}
              className="text-primary underline hover:opacity-90 dark:hover:text-foreground"
            >
              Terms of Service
            </Link>
          </p>
        </CardFooter>
      </Card>
    </div>
  );
}
