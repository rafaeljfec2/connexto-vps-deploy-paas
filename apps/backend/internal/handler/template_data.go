package handler

const (
	labelRootPassword  = "Root Password"
	labelUsername      = "Username"
	labelPassword      = "Password"
	labelDatabaseName  = "Database Name"
	labelAdminEmail    = "Admin Email"
	labelAdminPassword = "Admin Password"
	labelAdminUsername = "Admin Username"
	labelRootUsername  = "Root Username"

	categoryDatabase    = "database"
	categoryWebserver   = "webserver"
	categoryDevelopment = "development"
	categoryMonitoring  = "monitoring"
	categoryMessaging   = "messaging"
	categoryStorage     = "storage"

	templateTypeContainer = "container"
)

type Template struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Image       string           `json:"image"`
	Category    string           `json:"category"`
	Type        string           `json:"type"`
	Logo        string           `json:"logo,omitempty"`
	Env         []TemplateEnvVar `json:"env,omitempty"`
	Ports       []int            `json:"ports,omitempty"`
	Volumes     []string         `json:"volumes,omitempty"`
}

type TemplateEnvVar struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
}

var templates = []Template{
	{
		ID:          "postgres",
		Name:        "PostgreSQL",
		Description: "The most advanced open-source relational database",
		Image:       "postgres:16-alpine",
		Category:    categoryDatabase,
		Type:        templateTypeContainer,
		Logo:        "https://www.postgresql.org/media/img/about/press/elephant.png",
		Env: []TemplateEnvVar{
			{Name: "POSTGRES_USER", Label: labelUsername, Default: "postgres", Required: true},
			{Name: "POSTGRES_PASSWORD", Label: labelPassword, Required: true},
			{Name: "POSTGRES_DB", Label: labelDatabaseName, Default: "postgres", Required: false},
		},
		Ports:   []int{5432},
		Volumes: []string{"/var/lib/postgresql/data"},
	},
	{
		ID:          "mysql",
		Name:        "MySQL",
		Description: "Popular open-source relational database management system",
		Image:       "mysql:8",
		Category:    categoryDatabase,
		Type:        templateTypeContainer,
		Logo:        "https://www.mysql.com/common/logos/logo-mysql-170x115.png",
		Env: []TemplateEnvVar{
			{Name: "MYSQL_ROOT_PASSWORD", Label: labelRootPassword, Required: true},
			{Name: "MYSQL_DATABASE", Label: labelDatabaseName, Required: false},
			{Name: "MYSQL_USER", Label: labelUsername, Required: false},
			{Name: "MYSQL_PASSWORD", Label: labelPassword, Required: false},
		},
		Ports:   []int{3306},
		Volumes: []string{"/var/lib/mysql"},
	},
	{
		ID:          "mongodb",
		Name:        "MongoDB",
		Description: "Document-oriented NoSQL database",
		Image:       "mongo:7",
		Category:    categoryDatabase,
		Type:        templateTypeContainer,
		Logo:        "https://www.mongodb.com/assets/images/global/leaf.png",
		Env: []TemplateEnvVar{
			{Name: "MONGO_INITDB_ROOT_USERNAME", Label: labelRootUsername, Default: "admin", Required: true},
			{Name: "MONGO_INITDB_ROOT_PASSWORD", Label: labelRootPassword, Required: true},
		},
		Ports:   []int{27017},
		Volumes: []string{"/data/db"},
	},
	{
		ID:          "redis",
		Name:        "Redis",
		Description: "In-memory data structure store, cache, and message broker",
		Image:       "redis:7-alpine",
		Category:    categoryDatabase,
		Type:        templateTypeContainer,
		Logo:        "https://redis.io/images/redis-white.png",
		Ports:       []int{6379},
		Volumes:     []string{"/data"},
	},
	{
		ID:          "nginx",
		Name:        "Nginx",
		Description: "High-performance HTTP server and reverse proxy",
		Image:       "nginx:alpine",
		Category:    categoryWebserver,
		Type:        templateTypeContainer,
		Logo:        "https://nginx.org/nginx.png",
		Ports:       []int{80, 443},
		Volumes:     []string{"/usr/share/nginx/html", "/etc/nginx/conf.d"},
	},
	{
		ID:          "caddy",
		Name:        "Caddy",
		Description: "Fast and extensible multi-platform HTTP/1-2-3 web server with automatic HTTPS",
		Image:       "caddy:alpine",
		Category:    categoryWebserver,
		Type:        templateTypeContainer,
		Logo:        "https://caddyserver.com/resources/images/caddy-logo.svg",
		Ports:       []int{80, 443},
		Volumes:     []string{"/data", "/config"},
	},
	{
		ID:          "rabbitmq",
		Name:        "RabbitMQ",
		Description: "Open-source message broker with management UI",
		Image:       "rabbitmq:3-management-alpine",
		Category:    categoryMessaging,
		Type:        templateTypeContainer,
		Logo:        "https://www.rabbitmq.com/img/logo-rabbitmq.svg",
		Env: []TemplateEnvVar{
			{Name: "RABBITMQ_DEFAULT_USER", Label: labelUsername, Default: "admin", Required: true},
			{Name: "RABBITMQ_DEFAULT_PASS", Label: labelPassword, Required: true},
		},
		Ports:   []int{5672, 15672},
		Volumes: []string{"/var/lib/rabbitmq"},
	},
	{
		ID:          "adminer",
		Name:        "Adminer",
		Description: "Database management tool in a single PHP file",
		Image:       "adminer:latest",
		Category:    categoryDevelopment,
		Type:        templateTypeContainer,
		Logo:        "https://www.adminer.org/static/designs/logo.png",
		Ports:       []int{8080},
	},
	{
		ID:          "pgadmin",
		Name:        "pgAdmin",
		Description: "Web-based administration tool for PostgreSQL",
		Image:       "dpage/pgadmin4:latest",
		Category:    categoryDevelopment,
		Type:        templateTypeContainer,
		Logo:        "https://www.pgadmin.org/static/COMPILED/assets/img/logo-right-128.png",
		Env: []TemplateEnvVar{
			{Name: "PGADMIN_DEFAULT_EMAIL", Label: labelAdminEmail, Required: true},
			{Name: "PGADMIN_DEFAULT_PASSWORD", Label: labelAdminPassword, Required: true},
		},
		Ports:   []int{80},
		Volumes: []string{"/var/lib/pgadmin"},
	},
	{
		ID:          "portainer",
		Name:        "Portainer",
		Description: "Docker and Kubernetes management UI",
		Image:       "portainer/portainer-ce:latest",
		Category:    categoryDevelopment,
		Type:        templateTypeContainer,
		Logo:        "https://www.portainer.io/hubfs/portainer-logo-black.svg",
		Ports:       []int{9000, 9443},
		Volumes:     []string{"/data", "/var/run/docker.sock:/var/run/docker.sock"},
	},
	{
		ID:          "prometheus",
		Name:        "Prometheus",
		Description: "Monitoring system and time series database",
		Image:       "prom/prometheus:latest",
		Category:    categoryMonitoring,
		Type:        templateTypeContainer,
		Logo:        "https://prometheus.io/assets/prometheus_logo_grey.svg",
		Ports:       []int{9090},
		Volumes:     []string{"/prometheus"},
	},
	{
		ID:          "grafana",
		Name:        "Grafana",
		Description: "Analytics and interactive visualization platform",
		Image:       "grafana/grafana:latest",
		Category:    categoryMonitoring,
		Type:        templateTypeContainer,
		Logo:        "https://grafana.com/static/img/menu/grafana2.svg",
		Env: []TemplateEnvVar{
			{Name: "GF_SECURITY_ADMIN_USER", Label: labelAdminUsername, Default: "admin", Required: false},
			{Name: "GF_SECURITY_ADMIN_PASSWORD", Label: labelAdminPassword, Default: "admin", Required: false},
		},
		Ports:   []int{3000},
		Volumes: []string{"/var/lib/grafana"},
	},
	{
		ID:          "minio",
		Name:        "MinIO",
		Description: "High-performance object storage compatible with S3 API",
		Image:       "minio/minio:latest",
		Category:    categoryStorage,
		Type:        templateTypeContainer,
		Logo:        "https://min.io/resources/img/logo.svg",
		Env: []TemplateEnvVar{
			{Name: "MINIO_ROOT_USER", Label: labelRootUsername, Default: "minioadmin", Required: true},
			{Name: "MINIO_ROOT_PASSWORD", Label: labelRootPassword, Required: true},
		},
		Ports:   []int{9000, 9001},
		Volumes: []string{"/data"},
	},
	{
		ID:          "mailhog",
		Name:        "MailHog",
		Description: "Email testing tool with web UI",
		Image:       "mailhog/mailhog:latest",
		Category:    categoryDevelopment,
		Type:        templateTypeContainer,
		Ports:       []int{1025, 8025},
	},
	{
		ID:          "elasticsearch",
		Name:        "Elasticsearch",
		Description: "Distributed search and analytics engine",
		Image:       "elasticsearch:8.11.0",
		Category:    categoryDatabase,
		Type:        templateTypeContainer,
		Logo:        "https://static-www.elastic.co/v3/assets/bltefdd0b53724fa2ce/blt36f2da8d650732a0/5d0823c3d8ff351753cbc99f/logo-elasticsearch-32-color.svg",
		Env: []TemplateEnvVar{
			{Name: "discovery.type", Label: "Discovery Type", Default: "single-node", Required: true},
			{Name: "xpack.security.enabled", Label: "Security Enabled", Default: "false", Required: false},
		},
		Ports:   []int{9200, 9300},
		Volumes: []string{"/usr/share/elasticsearch/data"},
	},
}

func getTemplates() []Template {
	return templates
}

func findTemplate(id string) *Template {
	for i := range templates {
		if templates[i].ID == id {
			return &templates[i]
		}
	}
	return nil
}
