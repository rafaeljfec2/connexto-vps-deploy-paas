import { GitHubLogoIcon } from "@radix-ui/react-icons";
import { API_ROUTES } from "@/constants/routes";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface GitHubConnectProps {
  readonly message?: string;
}

export function GitHubConnect({ message }: GitHubConnectProps) {
  const handleInstall = () => {
    window.open(API_ROUTES.GITHUB.INSTALL, "_blank", "noopener,noreferrer");
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <GitHubLogoIcon className="h-5 w-5" aria-hidden="true" />
          Connect GitHub
        </CardTitle>
        <CardDescription>
          {message ??
            "Install the FlowDeploy GitHub App to access your repositories."}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Button
          onClick={handleInstall}
          className="w-full"
          aria-label="Install GitHub App to connect your repositories"
        >
          <GitHubLogoIcon className="mr-2 h-4 w-4" aria-hidden="true" />
          Install GitHub App
        </Button>
      </CardContent>
    </Card>
  );
}
