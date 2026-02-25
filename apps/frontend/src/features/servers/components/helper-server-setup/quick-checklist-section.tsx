import { CheckCircle2 } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  CHECKLIST_BACKEND,
  CHECKLIST_PANEL,
  CHECKLIST_REMOTE,
} from "@/features/servers/data/helper-server-setup";

export function QuickChecklistSection() {
  return (
    <section aria-labelledby="checklist-heading">
      <Card>
        <CardHeader>
          <CardTitle id="checklist-heading" className="flex items-center gap-2">
            <CheckCircle2 className="h-5 w-5" />
            Quick checklist
          </CardTitle>
          <CardDescription>
            Use this before and after provisioning.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div>
            <h4 className="mb-2 font-medium">On the remote server</h4>
            <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
              {CHECKLIST_REMOTE.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
          <div>
            <h4 className="mb-2 font-medium">Backend</h4>
            <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
              {CHECKLIST_BACKEND.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
          <div>
            <h4 className="mb-2 font-medium">Panel</h4>
            <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
              {CHECKLIST_PANEL.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
        </CardContent>
      </Card>
    </section>
  );
}
