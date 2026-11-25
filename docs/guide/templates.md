# Templates

Templates deploy a ready-made application in one step. Open the [Applications page](http://localhost:8080/apps), click **New application** and pick **From a template**. Picking a template opens a short review: check its name, target and settings, then click **Deploy**. The name starts as the template's own (`gitea`), or the next free one (`gitea-2`) if an application on that target already uses it.

**checkship** then:

1. creates the application, with production and staging on the target you chose and the template's settings saved as environment variables (see [environments](/reference/applications#environments));
2. queues its first production [deployment](/reference/deployments), using the template's compose file as the source;
3. opens that deployment so you can follow it.

The application is like any other from then on: redeploy it, promote it, change its variables or delete it. **From scratch** in the same menu opens the usual form.

## What a template asks for

A template's settings come in four kinds:

- **Settings you must fill**: **Deploy** stays off until they are set. The template is marked **Needs setup** in the gallery.
- **Filled in for you**:
  - a random secret, generated in your browser, which you can copy or regenerate (copy gives the secret alone, even when the variable carries it inside a longer value, such as Mailpit's `admin:<password>`);
  - a value taken from the application's URL, such as the address the app should advertise.
- **Optional**: a switch with a sensible default.

All of them are saved as the application's variables, so you can read or change them later in its settings. Values taken from the URL are set when the application is created, for each environment from its own address (staging's from `<app>-staging.<host>`): if you rename the application or move it to another target, update them too.

When the target has no URL (a manual proxy), there is no URL to take values from, so those settings become ones you must fill.

## The templates

Twenty templates in two kinds, each one light: a single container, with its data in a named volume.

<div class="template-grid">
  <a href="#apps-you-open"><img class="template-logo" src="/template-logos/uptime-kuma.svg" alt=""><span>Uptime Kuma<small>Monitoring</small></span></a>
  <a href="#apps-you-open"><img class="template-logo light-only" src="/template-logos/memos.svg" alt=""><img class="template-logo dark-only" src="/template-logos/memos-dark.svg" alt=""><span>Memos<small>Notes</small></span></a>
  <a href="#apps-you-open"><img class="template-logo" src="/template-logos/gitea.svg" alt=""><span>Gitea<small>Developer tools</small></span></a>
  <a href="#apps-you-open"><img class="template-logo light-only" src="/template-logos/vaultwarden.svg" alt=""><img class="template-logo dark-only" src="/template-logos/vaultwarden-white.svg" alt=""><span>Vaultwarden<small>Security</small></span></a>
  <a href="#apps-you-open"><img class="template-logo" src="/template-logos/gotify.svg" alt=""><span>Gotify<small>Notifications</small></span></a>
</div>

<div class="template-grid">
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/postgresql.svg" alt=""><span>PostgreSQL<small>Database</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo light-only" src="/template-logos/mariadb.svg" alt=""><img class="template-logo dark-only" src="/template-logos/mariadb-white.svg" alt=""><span>MariaDB<small>Database</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo plate" src="/template-logos/valkey.svg" alt=""><span>Valkey<small>Cache</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/pgadmin.svg" alt=""><span>pgAdmin<small>Database UI</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/rabbitmq.svg" alt=""><span>RabbitMQ<small>Messaging</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/nats.svg" alt=""><span>NATS<small>Messaging</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo light-only" src="/template-logos/mailpit.svg" alt=""><img class="template-logo dark-only" src="/template-logos/mailpit-dark.svg" alt=""><span>Mailpit<small>Email</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/meilisearch.svg" alt=""><span>Meilisearch<small>Search</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/rustfs.svg" alt=""><span>RustFS<small>File storage</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/keycloak.svg" alt=""><span>Keycloak<small>Sign-in</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/openobserve.svg" alt=""><span>OpenObserve<small>Observability</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo light-only" src="/template-logos/directus.svg" alt=""><img class="template-logo dark-only" src="/template-logos/directus-white.svg" alt=""><span>Directus<small>Backend and API</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo plate" src="/template-logos/centrifugo.svg" alt=""><span>Centrifugo<small>Real-time</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/mosquitto.svg" alt=""><span>Mosquitto<small>MQTT</small></span></a>
  <a href="#services-for-your-own-apps"><img class="template-logo" src="/template-logos/phpmyadmin.svg" alt=""><span>phpMyAdmin<small>Database UI</small></span></a>
</div>

### Apps you open

| Template                                                                                                                                                                                   | What it is                                          | Before deploy                                                               | After deploy                                                                                                              |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| <img class="template-logo" src="/template-logos/uptime-kuma.svg" alt="">**Uptime Kuma**                                                                                                    | Uptime monitoring with alerts                       | Nothing                                                                     | Create the admin account on the first visit.                                                                              |
| <img class="template-logo light-only" src="/template-logos/memos.svg" alt=""><img class="template-logo dark-only" src="/template-logos/memos-dark.svg" alt="">**Memos**                    | Notes and short posts                               | Nothing (the instance URL comes from the app's URL)                         | Sign up: the first account runs the instance.                                                                             |
| <img class="template-logo" src="/template-logos/gitea.svg" alt="">**Gitea**                                                                                                                | Git hosting with pull requests, issues and packages | Nothing (the root URL comes from the app's URL)                             | Run the installer on the first visit to create the admin. SSH is off: clone over HTTPS.                                   |
| <img class="template-logo light-only" src="/template-logos/vaultwarden.svg" alt=""><img class="template-logo dark-only" src="/template-logos/vaultwarden-white.svg" alt="">**Vaultwarden** | A light server for Bitwarden apps                   | Nothing (an admin token is generated, the domain comes from the app's URL)  | Create your account in the web vault, then turn sign-ups off. The admin page is at `/admin`, opened with the admin token. |
| <img class="template-logo" src="/template-logos/gotify.svg" alt="">**Gotify**                                                                                                              | Push notifications from scripts and apps            | **An admin username and password** (at least 8 characters, or generate one) | Sign in with that admin. Gotify only reads it on its first start; change it later in Gotify.                              |

::: warning Vaultwarden needs HTTPS
Its web vault only works in a secure context: an `https` address, or a `*.localhost` one such as `http://vaultwarden.docker.localhost`. On a plain `http` target with another domain, the drawer warns you; use a target with an `https` URL.
:::

### Services for your own apps

What a backend or a frontend runs beside it while you build it: databases, a cache, messaging, email, search, file storage and sign-in. Each one is protected by a secret generated before it deploys, and your other applications on the same target [connect to it by name](#connect-from-your-apps).

| Template                                                                                                                                                                             | What your apps get                                                                          | Generated for you                                                                                           | Your apps connect to                                                                                            | Its page                                                                                             |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| <img class="template-logo" src="/template-logos/postgresql.svg" alt="">**PostgreSQL 18**                                                                                             | The relational database. Database and user are both `app`.                                  | The password                                                                                                | `postgres://app:<password>@postgres.<app>:5432/app`                                                             | None: it has no web page.                                                                            |
| <img class="template-logo light-only" src="/template-logos/mariadb.svg" alt=""><img class="template-logo dark-only" src="/template-logos/mariadb-white.svg" alt="">**MariaDB 12.3**  | The MySQL-compatible database. Database and user are both `app`.                            | The app password and the root password                                                                      | `mysql://app:<password>@mariadb.<app>:3306/app`                                                                 | None: it has no web page.                                                                            |
| <img class="template-logo plate" src="/template-logos/valkey.svg" alt="">**Valkey 9**                                                                                                | A Redis-compatible store for caches, sessions and queues, saved to disk (AOF).              | The password                                                                                                | `redis://default:<password>@valkey.<app>:6379/0`                                                                | None: it has no web page.                                                                            |
| <img class="template-logo" src="/template-logos/pgadmin.svg" alt="">**pgAdmin 9**                                                                                                    | A web interface for your PostgreSQL databases.                                              | The sign-in password (**you give your email**)                                                              | Nothing: you open it                                                                                            | pgAdmin, signed in with your email and the password.                                                 |
| <img class="template-logo" src="/template-logos/rabbitmq.svg" alt="">**RabbitMQ 4**                                                                                                  | A message broker for queues and background jobs.                                            | The password of user `app`                                                                                  | `amqp://app:<password>@rabbitmq.<app>:5672`                                                                     | The management console, signed in as `app`.                                                          |
| <img class="template-logo" src="/template-logos/nats.svg" alt="">**NATS 2**                                                                                                          | Fast publish, subscribe and request between services, with JetStream streams saved to disk. | The token                                                                                                   | `nats://<token>@nats.<app>:4222`                                                                                | None: it has no web page.                                                                            |
| <img class="template-logo light-only" src="/template-logos/mailpit.svg" alt=""><img class="template-logo dark-only" src="/template-logos/mailpit-dark.svg" alt="">**Mailpit**        | An SMTP server that keeps every email your apps send and delivers none.                     | The inbox password (user `admin`)                                                                           | `smtp://mailpit.<app>:1025`, with any username and password                                                     | The inbox, signed in as `admin`.                                                                     |
| <img class="template-logo" src="/template-logos/meilisearch.svg" alt="">**Meilisearch**                                                                                              | Instant, typo-tolerant search.                                                              | The master key                                                                                              | `http://meilisearch.<app>:7700`, with the master key                                                            | The search preview, which asks for a key.                                                            |
| <img class="template-logo" src="/template-logos/rustfs.svg" alt="">**RustFS**                                                                                                        | S3-compatible storage for uploads and files (MinIO's free images stopped in 2025).          | The access key and the secret key                                                                           | `http://rustfs.<app>:9000`, path-style, with both keys                                                          | The console, at `/rustfs/console/`, signed in with the same keys. The S3 API also gets its own port. |
| <img class="template-logo" src="/template-logos/keycloak.svg" alt="">**Keycloak 26**                                                                                                 | Sign-in for your apps: users, OpenID Connect and OAuth 2.0.                                 | The password of user `admin`; the public address is taken from the URL                                      | `http://keycloak.<app>:8080/realms/<realm>` (tokens are issued by the app's URL)                                | The admin console, signed in as `admin`.                                                             |
| <img class="template-logo" src="/template-logos/openobserve.svg" alt="">**OpenObserve 1.1**                                                                                          | Logs, metrics and traces from your apps over OpenTelemetry, with search and dashboards.     | The password (**you give your email**)                                                                      | `http://openobserve.<app>:5080/api/default` (OTLP over HTTP), signed in with your email and the password        | OpenObserve, signed in with your email.                                                              |
| <img class="template-logo light-only" src="/template-logos/directus.svg" alt=""><img class="template-logo dark-only" src="/template-logos/directus-white.svg" alt="">**Directus 12** | A backend for your frontend: data models, REST and GraphQL APIs, files and users.           | The admin password and the token secret (**you give your email**); the public address is taken from the URL | `http://directus.<app>:8055`                                                                                    | The Directus app, signed in with your email.                                                         |
| <img class="template-logo plate" src="/template-logos/centrifugo.svg" alt="">**Centrifugo 6**                                                                                        | Real-time messaging over WebSockets for your frontends, published from your backend.        | The admin password, the API key and the token signing key (**you give your frontend's origins**)            | `http://centrifugo.<app>:8000/api` with the API key; browsers at `wss://centrifugo.<host>/connection/websocket` | The admin page, signed in with the password.                                                         |
| <img class="template-logo" src="/template-logos/mosquitto.svg" alt="">**Mosquitto 2**                                                                                                | An MQTT broker for devices and real-time apps.                                              | The password of user `app`                                                                                  | `mqtt://app:<password>@mosquitto.<app>:1883`; browsers at the app's URL over WebSockets                         | None: its URL is MQTT over WebSockets.                                                               |
| <img class="template-logo" src="/template-logos/phpmyadmin.svg" alt="">**phpMyAdmin 5**                                                                                              | A web interface for your MariaDB and MySQL databases.                                       | Nothing: you sign in with a database's own user                                                             | Nothing: you open it                                                                                            | phpMyAdmin's login, with the server (such as `mariadb.db`), user and password.                       |

Notes:

- **Valkey** is the Linux Foundation's BSD-licensed fork of Redis: Redis clients and libraries work with it unchanged. Use the `default` user in the URL: `redis://:<password>@` sends an empty user name, which Valkey refuses.
- **PostgreSQL** and **MariaDB** wait until they accept connections before their deployment succeeds. PostgreSQL's data sits in `/var/lib/postgresql`, where PostgreSQL 18 and later keep one folder per major version. MariaDB's root password is for administration: give your app the `app` user.
- **pgAdmin** asks for your email before it deploys, since you sign in with it; the drawer checks it is an address, as pgAdmin refuses to start otherwise. Add your PostgreSQL app as a server with its address, such as `postgres.db`, port `5432`, user `app`.
- **RabbitMQ** keeps a fixed host name, `rabbitmq`, because it stores its data in a folder named after it: a redeploy finds its queues again.
- **NATS** needs the token on every connection; JetStream is on, with its streams in the volume.
- **Keycloak** runs in development mode, with its own database in the volume: fine to build and test sign-in, not for real users. Every token names the app's URL as its issuer (`KC_HOSTNAME`, taken from the URL), even when your backend asks Keycloak at `keycloak.<app>:8080`: there it gets the keys and the other back-channel addresses at that name, so it validates the tokens your frontend received.
- **OpenObserve** and **Directus** ask for your email before they deploy, since you sign in with it; the drawer checks it is an address. Both read their admin only on their first start.
- **Directus** keeps its data in SQLite, in a volume, with uploads in another: right to build against, not for a large production site.
- **Centrifugo** asks where your frontend runs (its allowed origins): browsers from anywhere else are refused. It keeps channels in memory, so it has no volume; your backend signs connection tokens with the token signing key.
- **Mosquitto** writes its configuration and password file on every start, from the variable: change the password in the variables, then redeploy. Anonymous clients are refused. Its URL speaks MQTT over WebSockets, for browsers, so opening it in a tab shows nothing.
- **phpMyAdmin** keeps nothing and asks for the server on its login page: give it your MariaDB app's address, such as `mariadb.db`, with that database's user and password.
- **Mailpit** accepts any SMTP login, so set your app's SMTP user and password to anything; nothing leaves Mailpit. It keeps the last 5,000 messages.
- **Meilisearch** runs in development mode, which keeps its search preview at the app's URL; the master key still guards every route. Give your frontend a search-only key made with the master key, never the master key itself.
- **A database's port is public.** PostgreSQL, MariaDB, Valkey, RabbitMQ's AMQP, NATS and Mailpit's SMTP use `tcp` ports, which the target also publishes, on the port shown on the deployment (`tcp://<host>:<port>`), so you can reach them from your own machine. That is why each one gets a password or a token.

Images are pinned to the versions the templates were checked with. Each template's drawer links to the project's website and documentation.

## Edit the compose file

Every application has a **Compose file** tab, between Deployments and Logs. It shows the compose file each environment runs, from its latest deployment:

- edit it there, then click **Save and deploy to Production** (or Staging): **checkship** queues a new deployment of the edited file and opens it. The previous deployment stays in Deployments, to redeploy;
- **Changed** marks an edit, and **Reset** goes back to what runs; **Copy** and **Download** take the file as it is in the editor;
- each environment keeps its own draft while you switch between them; an environment not deployed yet starts from the other one's file;
- the tab checks what would stop the file before it is sent: an empty file, tabs used for indentation (YAML only allows spaces), no top-level `services:`. Anything deeper shows in the deployment's log.

An environment deployed from Git or from an archive has its compose file there: the tab says so, and you can write a compose file to deploy instead.

A template's own compose file is in its deploy drawer, under **Compose file**: open it to change the image tag, a port or a volume before deploying, or **Use the template's file** to undo. The template's settings stay variables, which override the file's `environment`, and an app whose file still runs the template's image keeps its logo.

## Sign-in and connection

Many apps ask for something the first time you open them: Mailpit's inbox password, Keycloak's admin, RustFS's keys, Vaultwarden's admin token. **checkship** keeps all of it in one place for every app deployed from a template:

- the deploy drawer shows **How you sign in** before you deploy;
- once deployed, open the app and click **Sign-in and connection** in its header, or pick **Sign-in and connection** in its row menu on the Applications page (it opens the app's page at `?access`).

The drawer reads the values from the app's own variables, so it shows what the app actually runs with, for production or staging:

- **Open it**: the app's address, then what its page asks for (user name, password, token, admin page). Secrets are masked; the eye button shows one, the copy button copies it whole.
- **Sign in with**, for a service without a web page: its password or token.
- **Connect from your apps**: the address other apps use, the connection string (only its secret masked), and the address **from outside the target**, for a tool on your own machine.
- A note for apps that read their sign-in only on their first start (Gotify, PostgreSQL, MariaDB, pgAdmin, RabbitMQ, Keycloak, OpenObserve, Directus): change the account in the app afterwards, as a new variable does nothing then.
- **Edit the variables** opens the app's settings.

Apps that create their account in the app (Uptime Kuma, Memos, Gitea) say so: there is nothing to enter from **checkship**. phpMyAdmin signs in with a database's own user.

## A service without a web page

A database or a broker (PostgreSQL, MariaDB, Valkey, NATS, Mosquitto) has no web address, only a `tcp` port. Its environment card on the app's page shows where it is instead of "No URL":

- **From outside**: `tcp://<target host>:<port>`, the port the target published, for a database tool on your machine;
- **From your apps**: `<service>.<app>:<port>`, the address on the target's network.

Each has a copy button. The **Visit** button stays off for such an app, and its tooltip says so.

## Connect from your apps

Every exposed service joins the target's network, where it answers at `<service>.<app>` (`<service>.<app>-staging` in staging). An application named `db` from the PostgreSQL template is `postgres.db:5432` for your other applications on that target. See [reaching a service from another application](/reference/providers/docker#reaching-a-service-from-another-application).

For a service template, the drawer shows a **Connect from your apps** section before you deploy: the address and a connection string with the generated secret, both with a copy button (the secret is masked on screen, never in what you copy). Paste the string into your backend's variables, for example `DATABASE_URL`.

- Your application must **expose a port** to be on the target's network: a web service does; a worker without ports doesn't.
- Use the **container port** shown (`5432`, `6379`…), not the one the target published.
- The address follows the name: renaming the application changes it.

## In your applications list

An application that runs a template's image shows that project's logo in the Applications list and beside its name on its page; its badge still says how its last deployment went, and the logo gives way to a spinner while it deploys or is deleted. **checkship** recognises it from its latest deployment's compose file when that file runs a single image whose repository is a template's, so an application you deployed from scratch with `postgres:17` alone shows PostgreSQL's logo too. One that runs several images, builds its own, or comes from Git shows the usual state icon.

A service without a web page, such as PostgreSQL, MariaDB, Valkey or NATS, says **No web page** in the list rather than showing an address that would open nothing, and its page has no Visit link to a `tcp://` address.

## Logos

Each template shows its project's own logo, never recoloured. The logos are trademarks of their projects and only identify each app. Their sources and licenses are listed in [`static/template-logos/NOTICE.md`](https://github.com/checkship-deploy/checkship/blob/main/cmd/serve/front/static/template-logos/NOTICE.md).

- Gotify's logo is licensed under CC BY 4.0, and the drawer credits it.
- PostgreSQL's elephant is a trademark of the PostgreSQL Community Association, used unmodified to refer to PostgreSQL, as its trademark policy allows.
- MariaDB's logotype is licensed under CC BY-SA 4.0 and attributed to the MariaDB Foundation. NATS's and Keycloak's come from the CNCF's artwork, under the Linux Foundation's trademark guidelines.
- Where a mark disappears on one theme, the project's own other file is used: Vaultwarden's white icon and MariaDB's white logotype in dark mode, Memos' and Mailpit's colours as each file declares them. Directus's white file shows in dark mode. Valkey and Centrifugo publish no light variant of their dark marks, so dark mode draws them on a white plate.
- This page shows the same files, from `docs/public/template-logos/`; the catalog tests keep both copies equal.

## Adding a template

Templates live in [`cmd/serve/front/src/lib/templates/catalog.ts`](https://github.com/checkship-deploy/checkship/blob/main/cmd/serve/front/src/lib/templates/catalog.ts). A template's compose file is deployed as a raw source, so it must:

- use `image:` only, because a raw source has no build context;
- publish every port with a host part: a web port as `"<port>:<port>"`, so the service gets the application's URL, and a database's as `"<port>:<port>/tcp"`;
- keep its data in named volumes, because bind mounts are not kept between deployments;
- set `restart: unless-stopped`.

Every value a person or **checkship** chooses goes in the template's `fields`, never as `${VAR}` in the compose file. **checkship** does not interpolate a compose file, but app variables override the compose file's `environment`. When an image only reads a secret inside a longer value, give the generated field a `prefix` (Valkey reads `VALKEY_EXTRA_FLAGS=--requirepass <password>`). A service other apps use sets `connect`: its port and a connection string with `{host}` and `{FIELD_KEY}` placeholders.

Add the logo under `static/template-logos/` with an entry in `NOTICE.md` (and `logoDark` or `logoPlate` if it disappears on a dark ground), and the template's text in every language file. The catalog tests check all of this.
