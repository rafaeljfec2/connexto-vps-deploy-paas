import { useEffect, useRef, useState } from "react";
import { AlertCircle, Check, Copy, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  type CreateTokenResponse,
  TOKEN_SCOPES,
  type TokenScope,
} from "@/services/api/tokens";
import { useCreateToken } from "../hooks/use-tokens";
import { SCOPE_DESCRIPTIONS } from "./scope-descriptions";

interface CreateTokenDialogProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

type FormStep = "form" | "reveal";

export function CreateTokenDialog({
  open,
  onOpenChange,
}: Readonly<CreateTokenDialogProps>) {
  const [step, setStep] = useState<FormStep>("form");
  const [name, setName] = useState("");
  const [selectedScopes, setSelectedScopes] = useState<Set<TokenScope>>(
    new Set<TokenScope>(["read"]),
  );
  const [expiryDays, setExpiryDays] = useState<number>(90);
  const [formError, setFormError] = useState<string | null>(null);
  const [result, setResult] = useState<CreateTokenResponse | null>(null);
  const [copied, setCopied] = useState(false);
  const copiedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const createMutation = useCreateToken();

  useEffect(() => {
    return () => {
      if (copiedTimerRef.current !== null) {
        clearTimeout(copiedTimerRef.current);
      }
    };
  }, []);

  const resetAndClose = (nextOpen: boolean) => {
    if (!nextOpen) {
      setStep("form");
      setName("");
      setSelectedScopes(new Set<TokenScope>(["read"]));
      setExpiryDays(90);
      setFormError(null);
      setResult(null);
      setCopied(false);
    }
    onOpenChange(nextOpen);
  };

  const toggleScope = (scope: TokenScope) => {
    setSelectedScopes((prev) => {
      const next = new Set(prev);
      if (next.has(scope)) {
        next.delete(scope);
      } else {
        next.add(scope);
      }
      return next;
    });
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);

    const trimmed = name.trim();
    if (trimmed.length < 3 || trimmed.length > 120) {
      setFormError("Name must be between 3 and 120 characters.");
      return;
    }
    if (selectedScopes.size === 0) {
      setFormError("Select at least one scope.");
      return;
    }
    if (expiryDays < 1 || expiryDays > 365) {
      setFormError("Expiry must be between 1 and 365 days.");
      return;
    }

    const expiresAt = new Date(
      Date.now() + expiryDays * 24 * 60 * 60 * 1000,
    ).toISOString();

    try {
      const response = await createMutation.mutateAsync({
        name: trimmed,
        scopes: Array.from(selectedScopes),
        expiresAt,
      });
      setResult(response);
      setStep("reveal");
    } catch (error) {
      setFormError(
        error instanceof Error ? error.message : "Failed to create token.",
      );
    }
  };

  const handleCopy = async () => {
    if (!result) return;
    try {
      await navigator.clipboard.writeText(result.plaintextToken);
      setCopied(true);
      if (copiedTimerRef.current !== null) {
        clearTimeout(copiedTimerRef.current);
      }
      copiedTimerRef.current = setTimeout(() => {
        setCopied(false);
        copiedTimerRef.current = null;
      }, 2000);
    } catch {
      setFormError("Unable to copy to clipboard. Select and copy manually.");
    }
  };

  return (
    <Dialog open={open} onOpenChange={resetAndClose}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        {step === "form" ? (
          <form onSubmit={handleSubmit} className="space-y-4">
            <DialogHeader>
              <DialogTitle>Create personal access token</DialogTitle>
              <DialogDescription>
                Tokens grant API access with the scopes you choose. They are
                shown only once after creation.
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-2">
              <Label htmlFor="token-name">Name</Label>
              <Input
                id="token-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="ci-github, cursor-local, incident-bot"
                autoComplete="off"
                disabled={createMutation.isPending}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="token-expiry">Expires in (days)</Label>
              <Input
                id="token-expiry"
                type="number"
                min={1}
                max={365}
                value={expiryDays}
                onChange={(e) => setExpiryDays(Number(e.target.value) || 0)}
                disabled={createMutation.isPending}
              />
            </div>

            <fieldset className="space-y-2">
              <legend className="text-sm font-medium">Scopes</legend>
              <div className="space-y-2">
                {TOKEN_SCOPES.map((scope) => {
                  const id = `scope-${scope}`;
                  const checked = selectedScopes.has(scope);
                  return (
                    <label
                      key={scope}
                      htmlFor={id}
                      className="hover:bg-muted/50 flex cursor-pointer items-start gap-3 rounded-md border p-3"
                    >
                      <Checkbox
                        id={id}
                        checked={checked}
                        onCheckedChange={() => toggleScope(scope)}
                        disabled={createMutation.isPending}
                      />
                      <div className="space-y-1 text-sm">
                        <div className="font-medium">{scope}</div>
                        <div className="text-muted-foreground text-xs">
                          {SCOPE_DESCRIPTIONS[scope]}
                        </div>
                      </div>
                    </label>
                  );
                })}
              </div>
            </fieldset>

            {formError ? (
              <p className="text-destructive flex items-center gap-2 text-sm">
                <AlertCircle className="h-4 w-4" />
                {formError}
              </p>
            ) : null}

            <DialogFooter className="gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => resetAndClose(false)}
                disabled={createMutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : null}
                Create token
              </Button>
            </DialogFooter>
          </form>
        ) : (
          <div className="space-y-4">
            <DialogHeader>
              <DialogTitle>Copy your token</DialogTitle>
              <DialogDescription>
                This is the only time you will see the full token. Store it in a
                secure secret manager immediately.
              </DialogDescription>
            </DialogHeader>

            <div className="bg-muted space-y-2 rounded-md border p-3">
              <div className="text-muted-foreground text-xs uppercase tracking-wide">
                Plaintext token
              </div>
              <code className="block break-all font-mono text-sm">
                {result?.plaintextToken}
              </code>
            </div>

            <Button
              type="button"
              onClick={handleCopy}
              className="w-full"
              variant="secondary"
            >
              {copied ? (
                <Check className="mr-2 h-4 w-4" />
              ) : (
                <Copy className="mr-2 h-4 w-4" />
              )}
              {copied ? "Copied" : "Copy to clipboard"}
            </Button>

            <DialogFooter>
              <Button onClick={() => resetAndClose(false)} className="w-full">
                I saved it, close
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
