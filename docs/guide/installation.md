# Installation

If you choose a Docker-based installation, you **must** have [Docker >= v18.0.9](https://docs.docker.com/get-docker/) installed (2019-09-03).

**Compose** and **Git** are packaged inside checkship itself so you don't have to bother with them.

## With Compose (recommended) {#with-compose}

Simply save the following `compose.yml` file in a folder, [configure it according to your needs](/guide/configuration) and run `docker compose up -d`:

```yml
services:
  web:
    restart: unless-stopped
    image: ghcr.io/checkship-deploy/checkship:latest # <- You probably better sets the version explicitly here
    environment:
      - ADMIN_EMAIL=admin@example.com # <- Change this
      - ADMIN_PASSWORD=admin # <- Change this
      # (...) sets other environment variables here, see the configuration documentation
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ssh:/root/.ssh # If you deploy on remote servers, keep ssh related configurations
      - data:/checkship/data # The /checkship/data directory contains the database, configuration file and everything deployed by checkship, so keep it :)

volumes:
  ssh:
  data:
```

With the Docker socket mounted as above, **checkship** creates a local target if needed and pulls and starts its built-in VersityGW backup storage on first start. You do not need `EXPOSED_ON` to enable backup storage; that setting is only for exposing **checkship** through its proxy. If the Docker engine cannot be reached, checkship still starts and keeps backup archives in its data folder.

## With Docker

Don't forget to sets the [appropriate environment variables](/guide/configuration) according to your needs.

```sh
docker run -d \
  -e "ADMIN_EMAIL=admin@example.com" \
  -e "ADMIN_PASSWORD=admin" \
  -v "/var/run/docker.sock:/var/run/docker.sock" \
  -v "checkshipdata:/checkship/data" \
  -v "checkshipssh:/root/.ssh" \
  -p "8080:8080" \
  ghcr.io/checkship-deploy/checkship
```

::: info
On windows, you may have to run the command on one line without the backslashes.
:::

## From sources

You'll need **Go** installed on your machine (at least 1.21) and **Node** for the frontend stuff (at least 1.18).

::: info
On **Windows**, you will need a gcc compiler such as [tdm gcc](https://jmeubank.github.io/tdm-gcc/) to build the sqlite3 driver correctly.
:::

Retrieve the checkship sources by either [cloning the repository](https://github.com/checkship-deploy/checkship) or downloading them on the [Releases page](https://github.com/checkship-deploy/checkship/releases) and run:

```sh
make build && ./checkship serve
```

## Exposing checkship itself {#exposing-checkship}

You probably want to expose checkship itself on an url without a port and with a valid certificate. You can manage this part yourself or just leverage the proxy deployed by checkship.

::: warning
If you choose to expose **checkship** with the local target, you **should never** delete the target or things may break.

If it happens, you can map the port `8080` to access your checkship instance (on the root domain and port) and reconfigure the target. Once done, you can remove the ports section since checkship will be exposed by the proxy.
:::

To do this, we recommend to use the [docker compose installation](#with-compose) and add the missing parts.

```yml
services:
  web:
    restart: unless-stopped
    image: ghcr.io/checkship-deploy/checkship:latest # <- You probably better sets the version explicitly here
    container_name: checkship # Should match the user portion of EXPOSED_ON when exposing checkship using a local target // [!code ++]
    environment:
      - ADMIN_EMAIL=admin@example.com # <- Change this
      - ADMIN_PASSWORD=admin # <- Change this
      # (...) sets other environment variables here, see the configuration documentation
      - EXPOSED_ON=http://checkship@docker.localhost # <- Change this to where you want checkship to be exposed, the user part represents the container name, see above. // [!code ++]
      - HTTP_SECURE= # Force fallback to the default handling of http secure (based on EXPOSED_ON if set) // [!code ++]
    ports: // [!code --]
      - "8080:8080" // [!code --]
    labels: // [!code ++]
      - app.checkship.exposed=true // [!code ++]
      - app.checkship.subdomain=mycheckship # Subdomain where checkship will be exposed on the default target represented by EXPOSED_ON, here http://mycheckship.docker.localhost // [!code ++]
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ssh:/root/.ssh # If you deploy on remote servers, keep ssh related configurations
      - data:/checkship/data # The /checkship/data directory contains the database, configuration file and everything deployed by checkship, so keep it :)

volumes:
  ssh:
  data:
```

::: info
If set, the `EXPOSED_ON` will be used to determine:

- The name of the checkship container itself which must be attached to the local target,
- The default local target URL if no one exists yet.

If a local target already exists, the container will be attached to it without updating the target URL. The local target may already have been created for built-in backup storage; set `EXPOSED_ON` before first startup if you want it to receive the default URL automatically.
:::

## Behind a reverse proxy {#reverse-proxy}

If you put checkship behind your own reverse proxy (nginx, Caddy, Traefik…), it must let **server-sent events** through as they are written. The dashboard uses them for live updates, the [live deployment logs](/reference/deployments#live-logs) and the runtime logs of an application. A proxy that buffers or compresses these responses makes logs arrive in bursts, or only when the deployment ends.

checkship already answers these streams with `Content-Type: text/event-stream`, `Cache-Control: no-store` and `X-Accel-Buffering: no`. Depending on your proxy, also:

::: code-group

```nginx [nginx]
location /api/v1/ {
    proxy_pass http://checkship:8080;
    proxy_http_version 1.1;
    proxy_buffering off;       # stream responses as they come
    proxy_cache off;
    proxy_read_timeout 1h;     # streams stay open while a deployment runs
    gzip off;
}
```

```caddy [Caddy]
checkship.example.com {
    # Compress only these types: text/event-stream stays out, compression breaks streaming
    encode {
        gzip
        match {
            header Content-Type text/html*
            header Content-Type text/css*
            header Content-Type application/javascript*
            header Content-Type application/json*
        }
    }
    reverse_proxy checkship:8080
}
```

```yml [Traefik (labels)]
labels:
  # Only if you add a compress middleware: leave event streams out of it
  - traefik.http.middlewares.checkship-compress.compress.excludedcontenttypes=text/event-stream
```

:::

The proxy deployed by checkship on its targets needs nothing: it doesn't compress or buffer.
