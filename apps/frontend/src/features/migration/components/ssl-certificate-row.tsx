import { Lock } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { SSLCertificate } from "@/types";

interface SSLCertificateRowProps {
  readonly certificate: SSLCertificate;
}

export function SSLCertificateRow({ certificate }: SSLCertificateRowProps) {
  const isExpiringSoon = certificate.daysUntilExpiry <= 30;
  const isExpired = certificate.isExpired;

  const iconColor = isExpired
    ? "text-status-failed"
    : isExpiringSoon
      ? "text-status-pending"
      : "text-status-success";

  const badgeVariant = isExpired
    ? "destructive"
    : isExpiringSoon
      ? "outline"
      : "secondary";

  return (
    <div className="flex items-center justify-between p-3 border rounded-lg">
      <div className="flex items-center gap-3">
        <Lock className={`h-5 w-5 ${iconColor}`} />
        <div>
          <p className="font-medium">{certificate.domain}</p>
          <p className="text-sm text-muted-foreground">
            Provider: {certificate.provider}
            {certificate.autoRenew && " • Auto-renew enabled"}
          </p>
        </div>
      </div>
      <div className="text-right">
        <Badge variant={badgeVariant}>
          {isExpired ? "Expired" : `${certificate.daysUntilExpiry} days left`}
        </Badge>
        <p className="text-xs text-muted-foreground mt-1">
          {new Date(certificate.expiresAt).toLocaleDateString()}
        </p>
      </div>
    </div>
  );
}
