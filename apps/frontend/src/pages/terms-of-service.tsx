import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";

const LAST_UPDATED = "2025-02-04";

export function TermsOfServicePage() {
  return (
    <div className="bg-background">
      <div className="container mx-auto max-w-3xl px-4 py-10 sm:px-6 sm:py-12">
        <Link
          to={ROUTES.LOGIN}
          className="text-muted-foreground hover:text-foreground mb-6 inline-block text-sm transition-colors"
        >
          ← Back to sign in
        </Link>

        <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">
          Terms of Service
        </h1>
        <p className="text-muted-foreground mt-2 text-sm">
          Last updated: {LAST_UPDATED}
        </p>

        <div className="prose prose-slate dark:prose-invert mt-10 max-w-none space-y-8">
          <section>
            <h2 className="text-xl font-semibold">1. Acceptance</h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              By signing in or using flowDeploy (“Service”), you agree to these
              Terms of Service (“Terms”). If you are using the Service on behalf
              of an organization, you represent that you have authority to bind
              that organization. If you do not agree, do not use the Service.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold">2. Description of Service</h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              flowDeploy is a self-hosted platform-as-a-service that enables
              deployment and management of applications (e.g., Git-based
              deployments, SSL, custom domains, logs, and rollbacks). You are
              responsible for hosting and operating the Service on your own
              infrastructure.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold">3. Your Obligations</h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              You agree to use the Service in compliance with applicable laws,
              not to misuse or abuse the Service or third-party services
              (including GitHub, Cloudflare, or your hosting provider), and to
              keep your account credentials secure. You are responsible for all
              activity under your account.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold">4. Account and Data</h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              Account data (e.g., profile from GitHub) is stored on the instance
              you control. You are responsible for backing up and securing your
              data. We do not access or store your data on our own servers when
              you self-host.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold">5. Acceptable Use</h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              You may not use the Service to violate laws, infringe rights,
              distribute malware, or conduct abuse (e.g., spam, unauthorized
              access). We may suspend or terminate access to the software or
              support if we believe you have violated these Terms.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold">6. Disclaimers</h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              The Service is provided “as is” without warranties of any kind. We
              do not guarantee uptime, security, or compatibility with your
              infrastructure. Use at your own risk.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold">
              7. Limitation of Liability
            </h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              To the maximum extent permitted by law, we are not liable for any
              indirect, incidental, special, or consequential damages arising
              from your use of the Service. Our total liability is limited to
              the amount you paid for the Service in the twelve months before
              the claim (if any).
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold">8. Changes</h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              We may update these Terms from time to time. We will post the
              updated Terms and update the “Last updated” date. Continued use of
              the Service after changes constitutes acceptance of the new Terms.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold">9. Contact</h2>
            <p className="text-muted-foreground mt-2 leading-relaxed">
              For questions about these Terms, please open an issue in the
              project repository or contact the maintainers through the official
              channel for the flowDeploy project.
            </p>
          </section>
        </div>

        <div className="mt-12 border-t border-border pt-6">
          <Link
            to={ROUTES.LOGIN}
            className="text-muted-foreground hover:text-foreground text-sm transition-colors"
          >
            ← Back to sign in
          </Link>
        </div>
      </div>
    </div>
  );
}
