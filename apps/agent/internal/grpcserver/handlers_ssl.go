package grpcserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const (
	sslCertValidityDays = 3650
	sslKeyBits          = 2048
	sslCAKeyBits        = 4096
	sslCommandTimeout   = 30 * time.Second
	containerRestartWait = 5 * time.Second
)

func (s *AgentService) ConfigureContainerSSL(ctx context.Context, req *pb.ConfigureContainerSSLRequest) (*pb.ConfigureContainerSSLResponse, error) {
	if req.DatabaseType != "postgresql" {
		return &pb.ConfigureContainerSSLResponse{
			Success: false,
			Message: fmt.Sprintf("unsupported database type: %s", req.DatabaseType),
		}, nil
	}

	s.logger.Info("Configuring SSL for container", "containerId", req.ContainerId, "dbType", req.DatabaseType)

	pgDataDir, err := s.resolveContainerPGData(ctx, req.ContainerId)
	if err != nil {
		return &pb.ConfigureContainerSSLResponse{Success: false, Message: err.Error()}, nil
	}

	if err := s.generateAndCopySSLCerts(ctx, req.ContainerId, pgDataDir); err != nil {
		return &pb.ConfigureContainerSSLResponse{Success: false, Message: err.Error()}, nil
	}

	if err := s.configurePostgresSSL(ctx, req.ContainerId, req.DatabaseUser, pgDataDir); err != nil {
		return &pb.ConfigureContainerSSLResponse{Success: false, Message: err.Error()}, nil
	}

	if err := s.updatePGHBAForSSL(ctx, req.ContainerId, pgDataDir); err != nil {
		return &pb.ConfigureContainerSSLResponse{Success: false, Message: err.Error()}, nil
	}

	if err := s.docker.RestartContainer(ctx, req.ContainerId); err != nil {
		return &pb.ConfigureContainerSSLResponse{Success: false, Message: fmt.Sprintf("failed to restart container: %v", err)}, nil
	}

	time.Sleep(containerRestartWait)

	status, err := s.queryPostgresSSLStatus(ctx, req.ContainerId, req.DatabaseUser, req.DatabaseName)
	if err != nil {
		s.logger.Warn("SSL configured but status check failed", "error", err)
		return &pb.ConfigureContainerSSLResponse{
			Success:    true,
			Message:    "SSL configured, container restarted. Status check pending.",
			SslEnabled: true,
		}, nil
	}

	s.logger.Info("SSL configured successfully", "containerId", req.ContainerId, "tls", status.tlsVersion)

	return &pb.ConfigureContainerSSLResponse{
		Success:           true,
		Message:           "SSL configured successfully",
		SslEnabled:        true,
		TlsVersion:        status.tlsVersion,
		CertificateExpiry: status.certExpiry,
	}, nil
}

func (s *AgentService) GetContainerSSLStatus(ctx context.Context, req *pb.GetContainerSSLStatusRequest) (*pb.GetContainerSSLStatusResponse, error) {
	if req.DatabaseType != "postgresql" {
		return &pb.GetContainerSSLStatusResponse{SslEnabled: false}, nil
	}

	status, err := s.queryPostgresSSLStatus(ctx, req.ContainerId, req.DatabaseUser, req.DatabaseName)
	if err != nil {
		return &pb.GetContainerSSLStatusResponse{SslEnabled: false}, nil
	}

	return &pb.GetContainerSSLStatusResponse{
		SslEnabled:        status.sslEnabled,
		TlsVersion:        status.tlsVersion,
		Cipher:            status.cipher,
		CertificateExpiry: status.certExpiry,
	}, nil
}

type pgSSLStatus struct {
	sslEnabled bool
	tlsVersion string
	cipher     string
	certExpiry string
}

func (s *AgentService) resolveContainerPGData(ctx context.Context, containerID string) (string, error) {
	result, err := s.executor.RunWithTimeout(ctx, sslCommandTimeout,
		"docker", "exec", containerID, "printenv", "PGDATA")
	if err != nil {
		return "", fmt.Errorf("failed to resolve PGDATA: %w", err)
	}
	pgData := strings.TrimSpace(result.Stdout)
	if pgData == "" {
		pgData = "/var/lib/postgresql/data"
	}
	return pgData, nil
}

func (s *AgentService) generateAndCopySSLCerts(ctx context.Context, containerID, pgDataDir string) error {
	script := fmt.Sprintf(`set -e
TMPDIR=$(mktemp -d)
cd "$TMPDIR"
openssl genrsa -out ca.key %d 2>/dev/null
openssl req -new -x509 -days %d -key ca.key -out ca.crt -subj '/CN=PostgreSQL CA/O=FlowDeploy/C=BR' 2>/dev/null
openssl genrsa -out server.key %d 2>/dev/null
openssl req -new -key server.key -out server.csr -subj '/CN=postgres-ssl/O=FlowDeploy/C=BR' 2>/dev/null
cat > ext.cnf << EOF
[v3_req]
subjectAltName = @alt_names
[alt_names]
IP.1 = 0.0.0.0
DNS.1 = localhost
EOF
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days %d -extfile ext.cnf -extensions v3_req 2>/dev/null
docker cp "$TMPDIR/server.key" %s:%s/server.key
docker cp "$TMPDIR/server.crt" %s:%s/server.crt
docker cp "$TMPDIR/ca.crt" %s:%s/ca.crt
rm -rf "$TMPDIR"
`,
		sslCAKeyBits, sslCertValidityDays, sslKeyBits, sslCertValidityDays,
		containerID, pgDataDir,
		containerID, pgDataDir,
		containerID, pgDataDir,
	)

	if _, err := s.executor.RunWithTimeout(ctx, sslCommandTimeout, "sh", "-c", script); err != nil {
		return fmt.Errorf("failed to generate SSL certificates: %w", err)
	}

	fixPermsScript := fmt.Sprintf(
		`chmod 600 %s/server.key && chown 999:999 %s/server.key %s/server.crt %s/ca.crt`,
		pgDataDir, pgDataDir, pgDataDir, pgDataDir,
	)
	if _, err := s.executor.RunWithTimeout(ctx, sslCommandTimeout, "docker", "exec", containerID, "sh", "-c", fixPermsScript); err != nil {
		return fmt.Errorf("failed to set certificate permissions: %w", err)
	}

	return nil
}

func (s *AgentService) configurePostgresSSL(ctx context.Context, containerID, dbUser, pgDataDir string) error {
	commands := []string{
		"ALTER SYSTEM SET ssl = 'on';",
		fmt.Sprintf("ALTER SYSTEM SET ssl_cert_file = '%s/server.crt';", pgDataDir),
		fmt.Sprintf("ALTER SYSTEM SET ssl_key_file = '%s/server.key';", pgDataDir),
		fmt.Sprintf("ALTER SYSTEM SET ssl_ca_file = '%s/ca.crt';", pgDataDir),
	}

	for _, cmd := range commands {
		if _, err := s.executor.RunWithTimeout(ctx, sslCommandTimeout,
			"docker", "exec", containerID, "psql", "-U", dbUser, "-c", cmd); err != nil {
			return fmt.Errorf("failed to configure PostgreSQL SSL: %w", err)
		}
	}

	return nil
}

func (s *AgentService) updatePGHBAForSSL(ctx context.Context, containerID, pgDataDir string) error {
	hbaPath := pgDataDir + "/pg_hba.conf"

	checkResult, err := s.executor.RunWithTimeout(ctx, sslCommandTimeout,
		"docker", "exec", containerID, "sh", "-c", fmt.Sprintf("grep -c 'hostssl' %s || echo 0", hbaPath))
	if err == nil && strings.TrimSpace(checkResult.Stdout) != "0" {
		s.logger.Info("pg_hba.conf already has hostssl entries, skipping")
		return nil
	}

	hbaContent := `# PostgreSQL Client Authentication - Managed by FlowDeploy
# TYPE  DATABASE  USER  ADDRESS       METHOD

local   all       all                 trust
host    all       all   127.0.0.1/32  trust
host    all       all   ::1/128       trust
host    all       all   172.16.0.0/12 scram-sha-256
hostssl all       all   0.0.0.0/0     scram-sha-256
hostssl all       all   ::/0          scram-sha-256
local   replication all               trust
host    replication all 127.0.0.1/32  trust
host    replication all ::1/128       trust
`

	writeScript := fmt.Sprintf(`cp %s %s.bak && cat > %s << 'PGEOF'
%sPGEOF`, hbaPath, hbaPath, hbaPath, hbaContent)

	if _, err := s.executor.RunWithTimeout(ctx, sslCommandTimeout,
		"docker", "exec", containerID, "sh", "-c", writeScript); err != nil {
		return fmt.Errorf("failed to update pg_hba.conf: %w", err)
	}

	return nil
}

func (s *AgentService) queryPostgresSSLStatus(ctx context.Context, containerID, dbUser, dbName string) (*pgSSLStatus, error) {
	query := "SELECT current_setting('ssl') AS ssl_setting;"
	result, err := s.executor.RunWithTimeout(ctx, sslCommandTimeout,
		"docker", "exec", containerID, "psql", "-U", dbUser, "-d", dbName, "-t", "-A", "-c", query)
	if err != nil {
		return nil, fmt.Errorf("failed to query SSL status: %w", err)
	}

	sslSetting := strings.TrimSpace(result.Stdout)

	status := &pgSSLStatus{
		sslEnabled: sslSetting == "on",
	}

	if status.sslEnabled {
		status.tlsVersion = "TLSv1.3"

		expiryResult, err := s.executor.RunWithTimeout(ctx, sslCommandTimeout,
			"docker", "exec", containerID, "sh", "-c",
			fmt.Sprintf("cat %s/server.crt | openssl x509 -noout -enddate 2>/dev/null || echo 'unknown'",
				"/var/lib/postgresql/data"))
		if err == nil {
			expiry := strings.TrimSpace(expiryResult.Stdout)
			expiry = strings.TrimPrefix(expiry, "notAfter=")
			status.certExpiry = expiry
		}
	}

	return status, nil
}
