import { ExternalLink, RefreshCw, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";

interface MobileActionBarProps {
  readonly onRedeploy: () => void;
  readonly onRollback: () => void;
  readonly openUrl?: string | null;
  readonly isRedeploying: boolean;
  readonly isRollingBack: boolean;
  readonly hasSuccessfulDeploy: boolean;
}

export function MobileActionBar({
  onRedeploy,
  onRollback,
  openUrl,
  isRedeploying,
  isRollingBack,
  hasSuccessfulDeploy,
}: MobileActionBarProps) {
  return (
    <div className="fixed bottom-0 inset-x-0 z-50 md:hidden bg-background/80 backdrop-blur-sm border-t px-4 py-2.5 pb-[calc(0.625rem+env(safe-area-inset-bottom))]">
      <div className="flex items-center gap-2 max-w-screen-sm mx-auto">
        <Button
          size="sm"
          className="flex-1"
          onClick={onRedeploy}
          disabled={isRedeploying}
        >
          <RefreshCw
            className={`h-4 w-4 mr-1.5 ${isRedeploying ? "animate-spin" : ""}`}
          />
          Redeploy
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="flex-1"
          onClick={onRollback}
          disabled={isRollingBack || !hasSuccessfulDeploy}
        >
          <RotateCcw className="h-4 w-4 mr-1.5" />
          Rollback
        </Button>
        {openUrl && (
          <Button variant="ghost" size="sm" asChild>
            <a href={openUrl} target="_blank" rel="noopener noreferrer">
              <ExternalLink className="h-4 w-4" />
            </a>
          </Button>
        )}
      </div>
    </div>
  );
}
