import { useState } from "react";
import { Play } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Template } from "@/types";
import { TemplateCategoryBadge } from "./template-category-badge";
import { TemplateDeployDialog } from "./template-deploy-dialog";

interface TemplateCardProps {
  readonly template: Template;
}

function TemplateLogo({ template }: { readonly template: Template }) {
  const [hasError, setHasError] = useState(false);

  if (!template.logo || hasError) {
    return (
      <div className="h-10 w-10 rounded bg-muted flex items-center justify-center text-lg font-bold">
        {template.name.charAt(0)}
      </div>
    );
  }

  return (
    <img
      src={template.logo}
      alt={template.name}
      className="h-10 w-10 object-contain"
      onError={() => setHasError(true)}
    />
  );
}

export function TemplateCard({ template }: TemplateCardProps) {
  const [showDeployDialog, setShowDeployDialog] = useState(false);

  return (
    <>
      <Card className="hover:bg-accent/50 transition-colors">
        <CardHeader className="pb-2">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-3">
              <TemplateLogo template={template} />
              <div>
                <CardTitle className="text-base">{template.name}</CardTitle>
                <TemplateCategoryBadge category={template.category} />
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-muted-foreground line-clamp-2">
            {template.description}
          </p>
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground font-mono">
              {template.image}
            </span>
            <Button size="sm" onClick={() => setShowDeployDialog(true)}>
              <Play className="h-4 w-4 mr-1" />
              Deploy
            </Button>
          </div>
        </CardContent>
      </Card>

      <TemplateDeployDialog
        template={template}
        open={showDeployDialog}
        onOpenChange={setShowDeployDialog}
      />
    </>
  );
}
