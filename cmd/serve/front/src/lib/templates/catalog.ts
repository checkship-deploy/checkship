import type { Template, TemplateCategory } from './types.ts';

/**
 * The built-in templates: light apps (one container each), each
 * with its project's own logo (static/template-logos/, sources and licenses in NOTICE.md). Images
 * are pinned to the versions checked on 2026-11-25; update them here.
 *
 * Every compose file follows what checkship's docker provider expects: `image:` only (a raw source
 * writes nothing but compose.yml), a `host:container` port so the service gets the app's URL, a
 * named volume (bind mounts are not kept), and a restart policy.
 */
export const templates: Template[] = [
	{
		id: 'uptime-kuma',
		name: 'Uptime Kuma',
		category: 'monitoring',
		description: 'templates.uptime_kuma.description',
		after: 'templates.uptime_kuma.after',
		image: 'louislam/uptime-kuma:2',
		service: 'uptime-kuma',
		compose: `services:
  uptime-kuma:
    image: louislam/uptime-kuma:2
    restart: unless-stopped
    environment:
      UPTIME_KUMA_DB_TYPE: sqlite
    ports:
      - "3001:3001"
    volumes:
      - data:/app/data

volumes:
  data:
`,
		fields: [],
		logo: '/template-logos/uptime-kuma.svg',
		logoCredit: 'Uptime Kuma (MIT)',
		website: 'https://uptime.kuma.pet',
		docs: 'https://github.com/louislam/uptime-kuma/wiki'
	},
	{
		id: 'memos',
		name: 'Memos',
		category: 'notes',
		description: 'templates.memos.description',
		after: 'templates.memos.after',
		image: 'neosmemo/memos:stable',
		service: 'memos',
		compose: `services:
  memos:
    image: neosmemo/memos:stable
    restart: unless-stopped
    environment:
      MEMOS_PORT: "5230"
    ports:
      - "5230:5230"
    volumes:
      - data:/var/opt/memos

volumes:
  data:
`,
		fields: [
			{
				key: 'MEMOS_INSTANCE_URL',
				kind: 'derived',
				derive: 'url',
				label: 'templates.memos.instance_url'
			}
		],
		logo: '/template-logos/memos.svg',
		logoDark: '/template-logos/memos-dark.svg',
		logoCredit: 'Memos (MIT)',
		website: 'https://usememos.com',
		docs: 'https://usememos.com/docs'
	},
	{
		id: 'gitea',
		name: 'Gitea',
		category: 'developer',
		description: 'templates.gitea.description',
		after: 'templates.gitea.after',
		image: 'docker.gitea.com/gitea:28.4.0',
		service: 'gitea',
		compose: `services:
  gitea:
    image: docker.gitea.com/gitea:28.4.0
    restart: unless-stopped
    environment:
      USER_UID: "1000"
      USER_GID: "1000"
      GITEA__server__DISABLE_SSH: "true"
    ports:
      - "3000:3000"
    volumes:
      - data:/data

volumes:
  data:
`,
		fields: [
			{
				key: 'GITEA__server__ROOT_URL',
				kind: 'derived',
				derive: 'url-slash',
				label: 'templates.gitea.root_url'
			}
		],
		logo: '/template-logos/gitea.svg',
		logoCredit: 'Gitea (MIT)',
		website: 'https://about.gitea.com',
		docs: 'https://docs.gitea.com'
	},
	{
		id: 'vaultwarden',
		name: 'Vaultwarden',
		category: 'security',
		description: 'templates.vaultwarden.description',
		after: 'templates.vaultwarden.after',
		needsSecureContext: true,
		image: 'vaultwarden/server:1.37.5',
		service: 'vaultwarden',
		compose: `services:
  vaultwarden:
    image: vaultwarden/server:1.37.5
    restart: unless-stopped
    ports:
      - "80:80"
    volumes:
      - data:/data

volumes:
  data:
`,
		fields: [
			{ key: 'DOMAIN', kind: 'derived', derive: 'url', label: 'templates.vaultwarden.domain' },
			{
				key: 'ADMIN_TOKEN',
				kind: 'generated',
				length: 48,
				label: 'templates.vaultwarden.admin_token'
			},
			{
				key: 'SIGNUPS_ALLOWED',
				kind: 'optional',
				default: true,
				label: 'templates.vaultwarden.signups',
				help: 'templates.vaultwarden.signups.help'
			}
		],
		credentials: [
			{ label: 'templates.credential.admin_page', value: '{url}/admin' },
			{ label: 'templates.credential.admin_token', value: '{ADMIN_TOKEN}', secret: true }
		],
		logo: '/template-logos/vaultwarden.svg',
		logoDark: '/template-logos/vaultwarden-white.svg',
		logoCredit: 'Vaultwarden (AGPL-3.0)',
		website: 'https://github.com/dani-garcia/vaultwarden',
		docs: 'https://github.com/dani-garcia/vaultwarden/wiki'
	},
	{
		id: 'gotify',
		name: 'Gotify',
		category: 'notifications',
		description: 'templates.gotify.description',
		after: 'templates.gotify.after',
		image: 'gotify/server:3.1.1',
		service: 'gotify',
		compose: `services:
  gotify:
    image: gotify/server:3.1.1
    restart: unless-stopped
    ports:
      - "80:80"
    volumes:
      - data:/app/data

volumes:
  data:
`,
		fields: [
			{
				key: 'GOTIFY_DEFAULTUSER_NAME',
				kind: 'required',
				label: 'templates.gotify.username',
				help: 'templates.gotify.username.help'
			},
			{
				key: 'GOTIFY_DEFAULTUSER_PASS',
				kind: 'required',
				secret: true,
				minLength: 8,
				length: 20,
				label: 'templates.gotify.password',
				help: 'templates.gotify.password.help'
			}
		],
		credentials: [
			{ label: 'templates.credential.username', value: '{GOTIFY_DEFAULTUSER_NAME}' },
			{ label: 'templates.credential.password', value: '{GOTIFY_DEFAULTUSER_PASS}', secret: true }
		],
		firstStartOnly: true,
		logo: '/template-logos/gotify.svg',
		logoCredit: 'Gotify (CC BY 4.0)',
		website: 'https://gotify.net',
		docs: 'https://gotify.net/docs'
	},

	// Services for the person's own applications: what a backend or frontend runs beside it. Each
	// publishes its port so checkship puts it on the target's network, where the person's other
	// applications reach it at <service>.<app> (see `connect`). A "/tcp" port is a custom
	// entrypoint: reachable from outside too, on the port the target assigns, so each one is
	// protected by a generated secret.
	{
		id: 'postgresql',
		name: 'PostgreSQL',
		category: 'database',
		description: 'templates.postgresql.description',
		after: 'templates.postgresql.after',
		image: 'postgres:18.7-alpine',
		service: 'postgres',
		// PostgreSQL 18 keeps its data under /var/lib/postgresql/18/docker: the volume sits one
		// level up, so a later major version can upgrade beside it.
		compose: `services:
  postgres:
    image: postgres:18.7-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: app
      POSTGRES_DB: app
    ports:
      - "5432:5432/tcp"
    volumes:
      - data:/var/lib/postgresql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d app"]
      interval: 5s
      timeout: 5s
      retries: 12

volumes:
  data:
`,
		fields: [
			{
				key: 'POSTGRES_PASSWORD',
				kind: 'generated',
				length: 32,
				label: 'templates.postgresql.password'
			}
		],
		connect: { port: 5432, url: 'postgres://app:{POSTGRES_PASSWORD}@{host}:5432/app' },
		credentials: [
			{ label: 'templates.credential.database', value: 'app' },
			{ label: 'templates.credential.username', value: 'app' },
			{ label: 'templates.credential.password', value: '{POSTGRES_PASSWORD}', secret: true }
		],
		firstStartOnly: true,
		logo: '/template-logos/postgresql.svg',
		logoCredit: 'PostgreSQL Community Association (trademark policy)',
		website: 'https://www.postgresql.org',
		docs: 'https://www.postgresql.org/docs/18/'
	},
	{
		id: 'valkey',
		name: 'Valkey',
		category: 'database',
		description: 'templates.valkey.description',
		after: 'templates.valkey.after',
		image: 'valkey/valkey:9.1.4-alpine',
		service: 'valkey',
		// The image appends VALKEY_EXTRA_FLAGS to the command after it drops to the valkey user,
		// which is how the password arrives without ${} interpolation.
		compose: `services:
  valkey:
    image: valkey/valkey:9.1.4-alpine
    restart: unless-stopped
    command: ["valkey-server", "--appendonly", "yes"]
    ports:
      - "6379:6379/tcp"
    volumes:
      - data:/data

volumes:
  data:
`,
		fields: [
			{
				key: 'VALKEY_EXTRA_FLAGS',
				kind: 'generated',
				length: 32,
				prefix: '--requirepass ',
				label: 'templates.valkey.password'
			}
		],
		connect: { port: 6379, url: 'redis://default:{VALKEY_EXTRA_FLAGS}@{host}:6379/0' },
		credentials: [
			{ label: 'templates.credential.password', value: '{VALKEY_EXTRA_FLAGS}', secret: true }
		],
		logo: '/template-logos/valkey.svg',
		logoPlate: true,
		logoCredit: 'Valkey (BSD-3-Clause)',
		website: 'https://valkey.io',
		docs: 'https://valkey.io/docs/'
	},
	{
		id: 'mailpit',
		name: 'Mailpit',
		category: 'email',
		description: 'templates.mailpit.description',
		after: 'templates.mailpit.after',
		image: 'axllent/mailpit:v1.34.0',
		service: 'mailpit',
		// It keeps every message and never delivers one: any SMTP login is accepted, so apps send
		// with whatever credentials they have.
		compose: `services:
  mailpit:
    image: axllent/mailpit:v1.34.0
    restart: unless-stopped
    environment:
      MP_DATABASE: /data/mailpit.db
      MP_MAX_MESSAGES: "5000"
      MP_SMTP_AUTH_ACCEPT_ANY: "true"
      MP_SMTP_AUTH_ALLOW_INSECURE: "true"
    ports:
      - "8025:8025"
      - "1025:1025/tcp"
    volumes:
      - data:/data

volumes:
  data:
`,
		fields: [
			{
				key: 'MP_UI_AUTH',
				kind: 'generated',
				length: 24,
				prefix: 'admin:',
				label: 'templates.mailpit.password'
			}
		],
		connect: { port: 1025, url: 'smtp://{host}:1025' },
		credentials: [
			{ label: 'templates.credential.username', value: 'admin' },
			{ label: 'templates.credential.password', value: '{MP_UI_AUTH}', secret: true }
		],
		logo: '/template-logos/mailpit.svg',
		logoDark: '/template-logos/mailpit-dark.svg',
		logoCredit: 'Mailpit (MIT)',
		website: 'https://mailpit.axllent.org',
		docs: 'https://mailpit.axllent.org/docs/'
	},
	{
		id: 'meilisearch',
		name: 'Meilisearch',
		category: 'search',
		description: 'templates.meilisearch.description',
		after: 'templates.meilisearch.after',
		image: 'getmeili/meilisearch:v1.58.0',
		service: 'meilisearch',
		// Development mode keeps the search preview at the app's URL (production mode answers
		// 404 there); the master key still guards every route.
		compose: `services:
  meilisearch:
    image: getmeili/meilisearch:v1.58.0
    restart: unless-stopped
    environment:
      MEILI_ENV: development
      MEILI_NO_ANALYTICS: "true"
    ports:
      - "7700:7700"
    volumes:
      - data:/meili_data

volumes:
  data:
`,
		fields: [
			{
				key: 'MEILI_MASTER_KEY',
				kind: 'generated',
				length: 48,
				label: 'templates.meilisearch.key'
			}
		],
		connect: { port: 7700, url: 'http://{host}:7700' },
		credentials: [
			{ label: 'templates.credential.master_key', value: '{MEILI_MASTER_KEY}', secret: true }
		],
		logo: '/template-logos/meilisearch.svg',
		logoCredit: 'Meilisearch (MIT Community Edition)',
		website: 'https://www.meilisearch.com',
		docs: 'https://www.meilisearch.com/docs'
	},
	{
		id: 'rustfs',
		name: 'RustFS',
		category: 'storage',
		description: 'templates.rustfs.description',
		after: 'templates.rustfs.after',
		image: 'rustfs/rustfs:1.0.6',
		service: 'rustfs',
		// The first web port gets the app's URL: the console, whose root sends a browser to
		// /rustfs/console/. The S3 API gets a port of its own on the target, and other apps reach
		// it at rustfs.<app>:9000.
		compose: `services:
  rustfs:
    image: rustfs/rustfs:1.0.6
    restart: unless-stopped
    environment:
      RUSTFS_VOLUMES: /data
      RUSTFS_ADDRESS: 0.0.0.0:9000
      RUSTFS_CONSOLE_ENABLE: "true"
      RUSTFS_CONSOLE_ADDRESS: 0.0.0.0:9001
    ports:
      - "9001:9001"
      - "9000:9000"
    volumes:
      - data:/data

volumes:
  data:
`,
		fields: [
			{
				key: 'RUSTFS_ACCESS_KEY',
				kind: 'generated',
				length: 20,
				label: 'templates.rustfs.access_key'
			},
			{
				key: 'RUSTFS_SECRET_KEY',
				kind: 'generated',
				length: 40,
				label: 'templates.rustfs.secret_key'
			}
		],
		connect: { port: 9000, url: 'http://{host}:9000' },
		credentials: [
			{ label: 'templates.credential.access_key', value: '{RUSTFS_ACCESS_KEY}' },
			{ label: 'templates.credential.secret_key', value: '{RUSTFS_SECRET_KEY}', secret: true }
		],
		logo: '/template-logos/rustfs.svg',
		logoCredit: 'RustFS (Apache-2.0)',
		website: 'https://rustfs.com',
		docs: 'https://docs.rustfs.com'
	},
	{
		id: 'mariadb',
		name: 'MariaDB',
		category: 'database',
		description: 'templates.mariadb.description',
		after: 'templates.mariadb.after',
		image: 'mariadb:12.3.3',
		service: 'mariadb',
		// healthcheck.sh ships in the image and reads the credentials its entrypoint writes on the
		// first start, so the deployment succeeds once the server takes connections.
		compose: `services:
  mariadb:
    image: mariadb:12.3.3
    restart: unless-stopped
    environment:
      MARIADB_USER: app
      MARIADB_DATABASE: app
    ports:
      - "3306:3306/tcp"
    volumes:
      - data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 24

volumes:
  data:
`,
		fields: [
			{
				key: 'MARIADB_PASSWORD',
				kind: 'generated',
				length: 32,
				label: 'templates.mariadb.password'
			},
			{
				key: 'MARIADB_ROOT_PASSWORD',
				kind: 'generated',
				length: 32,
				label: 'templates.mariadb.root_password'
			}
		],
		connect: { port: 3306, url: 'mysql://app:{MARIADB_PASSWORD}@{host}:3306/app' },
		credentials: [
			{ label: 'templates.credential.database', value: 'app' },
			{ label: 'templates.credential.username', value: 'app' },
			{ label: 'templates.credential.password', value: '{MARIADB_PASSWORD}', secret: true },
			{
				label: 'templates.credential.root_password',
				value: '{MARIADB_ROOT_PASSWORD}',
				secret: true
			}
		],
		firstStartOnly: true,
		logo: '/template-logos/mariadb.svg',
		logoDark: '/template-logos/mariadb-white.svg',
		logoCredit: 'MariaDB Foundation (CC BY-SA 4.0)',
		website: 'https://mariadb.org',
		docs: 'https://mariadb.com/docs/server'
	},
	{
		id: 'pgadmin',
		name: 'pgAdmin',
		category: 'database',
		description: 'templates.pgadmin.description',
		after: 'templates.pgadmin.after',
		image: 'dpage/pgadmin4:9.19',
		service: 'pgadmin',
		compose: `services:
  pgadmin:
    image: dpage/pgadmin4:9.19
    restart: unless-stopped
    ports:
      - "80:80"
    volumes:
      - data:/var/lib/pgadmin

volumes:
  data:
`,
		fields: [
			{
				key: 'PGADMIN_DEFAULT_EMAIL',
				kind: 'required',
				email: true,
				label: 'templates.pgadmin.email',
				help: 'templates.pgadmin.email.help'
			},
			{
				key: 'PGADMIN_DEFAULT_PASSWORD',
				kind: 'generated',
				length: 24,
				label: 'templates.pgadmin.password'
			}
		],
		credentials: [
			{ label: 'templates.credential.email', value: '{PGADMIN_DEFAULT_EMAIL}' },
			{ label: 'templates.credential.password', value: '{PGADMIN_DEFAULT_PASSWORD}', secret: true }
		],
		firstStartOnly: true,
		logo: '/template-logos/pgadmin.svg',
		logoCredit: 'pgAdmin Development Team (PostgreSQL License)',
		website: 'https://www.pgadmin.org',
		docs: 'https://www.pgadmin.org/docs/pgadmin4/latest/'
	},
	{
		id: 'rabbitmq',
		name: 'RabbitMQ',
		category: 'messaging',
		description: 'templates.rabbitmq.description',
		after: 'templates.rabbitmq.after',
		image: 'rabbitmq:4.4.1-management-alpine',
		service: 'rabbitmq',
		// RabbitMQ keeps its data under a folder named after the node, rabbit@<hostname>: a fixed
		// hostname keeps the same folder when a deployment recreates the container.
		compose: `services:
  rabbitmq:
    image: rabbitmq:4.4.1-management-alpine
    hostname: rabbitmq
    restart: unless-stopped
    environment:
      RABBITMQ_DEFAULT_USER: app
    ports:
      - "15672:15672"
      - "5672:5672/tcp"
    volumes:
      - data:/var/lib/rabbitmq

volumes:
  data:
`,
		fields: [
			{
				key: 'RABBITMQ_DEFAULT_PASS',
				kind: 'generated',
				length: 32,
				label: 'templates.rabbitmq.password'
			}
		],
		connect: { port: 5672, url: 'amqp://app:{RABBITMQ_DEFAULT_PASS}@{host}:5672' },
		credentials: [
			{ label: 'templates.credential.username', value: 'app' },
			{ label: 'templates.credential.password', value: '{RABBITMQ_DEFAULT_PASS}', secret: true }
		],
		firstStartOnly: true,
		logo: '/template-logos/rabbitmq.svg',
		logoCredit: 'RabbitMQ (Apache-2.0 website repository)',
		website: 'https://www.rabbitmq.com',
		docs: 'https://www.rabbitmq.com/docs'
	},
	{
		id: 'nats',
		name: 'NATS',
		category: 'messaging',
		description: 'templates.nats.description',
		after: 'templates.nats.after',
		image: 'nats:2.15.3-alpine',
		service: 'nats',
		// nats-server takes its token as a flag: the shell reads it from the app variable ($$ keeps
		// compose from reading it first), and exec leaves nats-server as the main process.
		compose: `services:
  nats:
    image: nats:2.15.3-alpine
    restart: unless-stopped
    command: ["sh", "-c", "exec nats-server --jetstream --store_dir /data --auth \\"$$NATS_TOKEN\\""]
    ports:
      - "4222:4222/tcp"
    volumes:
      - data:/data

volumes:
  data:
`,
		fields: [
			{
				key: 'NATS_TOKEN',
				kind: 'generated',
				length: 32,
				label: 'templates.nats.token'
			}
		],
		connect: { port: 4222, url: 'nats://{NATS_TOKEN}@{host}:4222' },
		credentials: [{ label: 'templates.credential.token', value: '{NATS_TOKEN}', secret: true }],
		logo: '/template-logos/nats.svg',
		logoCredit: 'NATS, CNCF (Linux Foundation trademark guidelines)',
		website: 'https://nats.io',
		docs: 'https://docs.nats.io'
	},
	{
		id: 'keycloak',
		name: 'Keycloak',
		category: 'auth',
		description: 'templates.keycloak.description',
		after: 'templates.keycloak.after',
		image: 'quay.io/keycloak/keycloak:26.8.2',
		service: 'keycloak',
		// Development mode keeps its database in /opt/keycloak/data. KC_HOSTNAME, the app's URL,
		// is the issuer of every token, wherever it is asked: a backend that fetches keys at
		// keycloak.<app>:8080 still validates tokens the browser got (back-channel URLs follow the
		// request, KC_HOSTNAME_BACKCHANNEL_DYNAMIC).
		compose: `services:
  keycloak:
    image: quay.io/keycloak/keycloak:26.8.2
    restart: unless-stopped
    command: ["start-dev"]
    environment:
      KC_BOOTSTRAP_ADMIN_USERNAME: admin
      KC_PROXY_HEADERS: xforwarded
      KC_HOSTNAME_BACKCHANNEL_DYNAMIC: "true"
    ports:
      - "8080:8080"
    volumes:
      - data:/opt/keycloak/data

volumes:
  data:
`,
		fields: [
			{
				key: 'KC_HOSTNAME',
				kind: 'derived',
				derive: 'url',
				label: 'templates.keycloak.hostname'
			},
			{
				key: 'KC_BOOTSTRAP_ADMIN_PASSWORD',
				kind: 'generated',
				length: 24,
				label: 'templates.keycloak.password'
			}
		],
		connect: { port: 8080, url: 'http://{host}:8080/realms/master' },
		credentials: [
			{ label: 'templates.credential.username', value: 'admin' },
			{
				label: 'templates.credential.password',
				value: '{KC_BOOTSTRAP_ADMIN_PASSWORD}',
				secret: true
			}
		],
		firstStartOnly: true,
		logo: '/template-logos/keycloak.svg',
		logoCredit: 'Keycloak, CNCF (Linux Foundation trademark guidelines)',
		website: 'https://www.keycloak.org',
		docs: 'https://www.keycloak.org/documentation'
	}
];

export const categories: TemplateCategory[] = [
	'monitoring',
	'notes',
	'developer',
	'security',
	'notifications',
	'database',
	'email',
	'search',
	'storage',
	'messaging',
	'auth'
];

export function templateById(id: string): Template | undefined {
	return templates.find((t) => t.id === id);
}
