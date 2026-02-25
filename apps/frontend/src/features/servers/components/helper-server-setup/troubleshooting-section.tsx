import { Terminal } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  TROUBLE_AGENT,
  TROUBLE_DOCKER,
  TROUBLE_MULTITENANCY,
  TROUBLE_SSH,
  TROUBLE_TRAEFIK,
} from "@/features/servers/data/helper-server-setup";
import { TroubleshootSection } from "./troubleshoot-section";

export function TroubleshootingSection() {
  return (
    <section aria-labelledby="troubleshooting-heading">
      <Card>
        <CardHeader>
          <CardTitle
            id="troubleshooting-heading"
            className="flex items-center gap-2"
          >
            <Terminal className="h-5 w-5" />
            Troubleshooting
          </CardTitle>
          <CardDescription>
            Common problems organized by category.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="ssh" className="w-full">
            <TabsList className="mb-4 flex flex-wrap h-auto gap-1">
              <TabsTrigger value="ssh" className="text-xs sm:text-sm">
                SSH
              </TabsTrigger>
              <TabsTrigger value="docker" className="text-xs sm:text-sm">
                Docker
              </TabsTrigger>
              <TabsTrigger value="traefik" className="text-xs sm:text-sm">
                Traefik
              </TabsTrigger>
              <TabsTrigger value="agent" className="text-xs sm:text-sm">
                Agent
              </TabsTrigger>
              <TabsTrigger value="multitenancy" className="text-xs sm:text-sm">
                Access
              </TabsTrigger>
            </TabsList>
            <TabsContent value="ssh">
              <TroubleshootSection items={TROUBLE_SSH} />
            </TabsContent>
            <TabsContent value="docker">
              <TroubleshootSection items={TROUBLE_DOCKER} />
            </TabsContent>
            <TabsContent value="traefik">
              <TroubleshootSection items={TROUBLE_TRAEFIK} />
            </TabsContent>
            <TabsContent value="agent">
              <TroubleshootSection items={TROUBLE_AGENT} />
            </TabsContent>
            <TabsContent value="multitenancy">
              <TroubleshootSection items={TROUBLE_MULTITENANCY} />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </section>
  );
}
