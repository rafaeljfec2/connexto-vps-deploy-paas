import { Shield } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FIREWALL_PORTS } from "@/features/servers/data/helper-server-setup";

export function FirewallPortsSection() {
  return (
    <section aria-labelledby="firewall-heading">
      <Card>
        <CardHeader>
          <CardTitle id="firewall-heading" className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Firewall ports
          </CardTitle>
          <CardDescription>
            Open these ports on the correct servers for provisioning and
            communication.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="overflow-x-auto rounded-md border">
            <table className="w-full min-w-[480px] text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-3 py-2 text-left font-medium">Port</th>
                  <th className="px-3 py-2 text-left font-medium">Protocol</th>
                  <th className="px-3 py-2 text-left font-medium">Purpose</th>
                  <th className="px-3 py-2 text-left font-medium">Required</th>
                  <th className="px-3 py-2 text-left font-medium">Command</th>
                </tr>
              </thead>
              <tbody className="text-muted-foreground">
                {FIREWALL_PORTS.map((p) => (
                  <tr key={p.port} className="border-b last:border-0">
                    <td className="px-3 py-2 font-medium text-foreground">
                      {p.port}
                    </td>
                    <td className="px-3 py-2">{p.protocol}</td>
                    <td className="px-3 py-2">{p.purpose}</td>
                    <td className="px-3 py-2">{p.required}</td>
                    <td className="px-3 py-2">
                      <code className="text-xs">{p.command}</code>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-sm text-muted-foreground">
            After adding rules, run <code>sudo ufw reload</code>. With
            firewalld:{" "}
            <code>sudo firewall-cmd --permanent --add-port=PORT/tcp</code> then{" "}
            <code>sudo firewall-cmd --reload</code>.
          </p>
        </CardContent>
      </Card>
    </section>
  );
}
