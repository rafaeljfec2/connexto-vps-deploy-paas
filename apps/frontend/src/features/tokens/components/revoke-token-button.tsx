import { useState } from "react";
import { Loader2, Trash2 } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { useRevokeToken } from "../hooks/use-tokens";

interface RevokeTokenButtonProps {
  readonly tokenId: string;
  readonly tokenName: string;
}

export function RevokeTokenButton({
  tokenId,
  tokenName,
}: Readonly<RevokeTokenButtonProps>) {
  const [open, setOpen] = useState(false);
  const revokeMutation = useRevokeToken();

  const handleConfirm = async () => {
    await revokeMutation.mutateAsync(tokenId);
    setOpen(false);
  };

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={`Revoke ${tokenName}`}>
          <Trash2 className="h-4 w-4" />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke token?</AlertDialogTitle>
          <AlertDialogDescription>
            Revoking <strong>{tokenName}</strong> immediately disables it. Any
            integration using this token will start receiving 401 Unauthorized
            responses.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={revokeMutation.isPending}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={revokeMutation.isPending}
          >
            {revokeMutation.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : null}
            Revoke token
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
