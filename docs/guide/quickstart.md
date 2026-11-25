# Quickstart

This quickstart installs **checkship** on your own computer and deploys a first application with it. On the way, you'll meet the resources **checkship** manages: targets, applications, deployments and jobs.

## What is checkship?

**checkship** is a self-hosted platform to deploy your **own applications** on your **own hardware**, from a web dashboard.

<img class="screenshot light-only" src="/screenshots/overview-light.webp" alt="The checkship Overview page with deployment charts" width="1440" height="900" />
<img class="screenshot dark-only" src="/screenshots/overview-dark.webp" alt="The checkship Overview page with deployment charts" width="1440" height="900" />

At its core, **checkship** reads a `compose.yml` file, **deploys** its services, and gives the ones [that publish a port](/reference/providers/docker#exposing-services) a **domain** with a **certificate**.

Most of the time, a `compose.yml` file that works on your machine is all you need, so **moving from a local stack to a remote one is a breeze**.

::: info
For now, only the [Docker provider](/reference/providers/docker) is available. Others, such as **Docker Swarm**, **Podman** or **Kubernetes**, may follow.
:::

Think of it as an alternative to services like [Heroku](https://www.heroku.com/), [Dokku](https://dokku.com/), [CapRover](https://caprover.com/) or [Coolify](https://coolify.io/).

Now that you know what **checkship** is, let's deploy an application of our own!

## Prerequisites

Since this quickstart targets your local Docker engine, the **only prerequisite is a Docker engine** (v18.09 or later) up and running.

The proxy **checkship** deploys listens on port **80**, so make sure nothing else uses it.

## Installing

For this quickstart, we'll run **checkship** on the local Docker engine. To host it on a server, follow [the installation guide](/guide/installation) instead.

Launching **checkship** takes one command:

```sh
docker run -d -e "ADMIN_EMAIL=admin@example.com" -e "ADMIN_PASSWORD=admin" -v "/var/run/docker.sock:/var/run/docker.sock" -p "8080:8080" ghcr.io/checkship-deploy/checkship
```

This starts a **checkship** instance and **creates an admin account** with these credentials **if no account exists yet** (otherwise they are ignored).

::: warning
This is a throwaway instance: no volume is attached, so everything **checkship** stores goes away with the container. In production, attach one as described in the [installation guide](/guide/installation).
:::

Open http://localhost:8080 and sign in with `admin@example.com` and `admin`, or **the credentials you set above**.

You land on the **Overview**. It stays empty until your first deployment and shows the three steps to get there.

<img class="screenshot light-only" src="/screenshots/first-run-light.webp" alt="The empty Overview page listing three steps: add a target, create an application, deploy it" width="1440" height="900" />
<img class="screenshot dark-only" src="/screenshots/first-run-dark.webp" alt="The empty Overview page listing three steps: add a target, create an application, deploy it" width="1440" height="900" />

## Create your first target

First, add [a target](/reference/targets): the Docker engine that will run your applications.

Click **Start** on the Overview, or open the [Targets page](http://localhost:8080/targets) and click [New target](http://localhost:8080/targets/new). Then fill the form:

- **Name**: `local`. It's only a label for lists and selects.
- **Where it runs**: **This machine**, the Docker engine next to **checkship**.
- **Exposure**: keep **Expose services automatically** on and set the **Root URL** to `http://docker.localhost`. Every application gets a subdomain of it, here `http://<app>.docker.localhost`. With an `https` URL, certificates are issued automatically.

<img class="screenshot light-only" src="/screenshots/new-target-light.webp" alt="The New target form with the name local, This machine selected and the root URL http://docker.localhost" width="1440" height="900" />
<img class="screenshot dark-only" src="/screenshots/new-target-dark.webp" alt="The New target form with the name local, This machine selected and the root URL http://docker.localhost" width="1440" height="900" />

Click **Create target**. The target starts as **Configuring** while **checkship** [prepares it](/reference/targets#configuration) and deploys the proxy that routes your application URLs (Traefik). After a few seconds, it turns **Ready**.

## Create your first application

Open the [Applications page](http://localhost:8080/apps), click **New application** and pick **From scratch**. (To deploy a ready-made app such as Uptime Kuma or Gitea in one step instead, pick **From a template**: see [Templates](/guide/templates).)

- **Name**: `welcome`. The name becomes part of every URL **checkship** exposes, so it must be **unique** on a target.
- **Version control**: leave **Use a Git repository** off. We'll upload an archive instead.
- **Environments**: production and staging both run on `local`, the only target.

<img class="screenshot light-only" src="/screenshots/new-app-light.webp" alt="The New application form with the name welcome and both environments on the local target" width="1440" height="900" />
<img class="screenshot dark-only" src="/screenshots/new-app-dark.webp" alt="The New application form with the name welcome and both environments on the local target" width="1440" height="900" />

Click **Create application**. To dig further, see the [applications reference](/reference/applications).

## Your first deployment

You're now on the application page. Click **Deploy** to open the **New deployment** drawer and queue your first [deployment](/reference/deployments). The sources it offers depend on the application: **Git** appears once version control is set.

We'll deploy the **SvelteKit application packaged as an archive** from the [examples directory](https://github.com/checkship-deploy/checkship/tree/main/examples/sveltekit-hello/) of the **checkship** source code.

1. [Download the archive](https://github.com/checkship-deploy/checkship/raw/main/examples/sveltekit-hello/sveltekit-hello.tar.gz).
2. In the drawer, keep **Production** as the environment and pick **Archive** as the source.
3. Drop the file you downloaded, or browse to it.

<img class="screenshot light-only" src="/screenshots/deploy-light.webp" alt="The New deployment drawer with Production and Archive selected and sveltekit-hello.tar.gz attached" width="1440" height="900" />
<img class="screenshot dark-only" src="/screenshots/deploy-dark.webp" alt="The New deployment drawer with Production and Archive selected and sveltekit-hello.tar.gz attached" width="1440" height="900" />

Click **Deploy**. A preview opens and follows the deployment live with its last log lines; **Open full page** shows its timeline, source, services and whole log. Depending on whether the base images are already on your machine, it takes a few seconds to a few minutes.

<img class="screenshot light-only" src="/screenshots/deployment-light.webp" alt="A succeeded deployment with its timeline, source, services and logs" width="1440" height="900" />
<img class="screenshot dark-only" src="/screenshots/deployment-dark.webp" alt="A succeeded deployment with its timeline, source, services and logs" width="1440" height="900" />

Once it **Succeeded**, open http://welcome.docker.localhost and see the application running!

Back on the application page, the **Overview** tab shows staging and production side by side. Try a change on staging first, then **Promote** it to production: production gets the same source with its own variables.

<img class="screenshot light-only" src="/screenshots/application-light.webp" alt="The application Overview with staging and production side by side and the Promote button between them" width="1440" height="900" />
<img class="screenshot dark-only" src="/screenshots/application-dark.webp" alt="The application Overview with staging and production side by side and the Promote button between them" width="1440" height="900" />

Congratulations 🎉, you have deployed your first application with **checkship**!

## Cleaning up resources

Before ending this quickstart, let's clean up everything we deployed.

Go back to **the application page** and open its **Settings** tab. In **Danger zone**, at the end, click **Delete application**, type its name and confirm. If an environment has successful deployments, a cleanup job removes everything the application created on its target.

<img class="screenshot light-only" src="/screenshots/danger-zone-light.webp" alt="The application Settings tab with Delete application in Danger zone" width="1440" height="900" />
<img class="screenshot dark-only" src="/screenshots/danger-zone-dark.webp" alt="The application Settings tab with Delete application in Danger zone" width="1440" height="900" />

::: danger
Deleting a resource in **checkship** is **permanent** and can't be undone. To keep the data of your volumes, [back them up](/reference/backups) **before**.
:::

You can follow the cleanup on the [Jobs page](http://localhost:8080/jobs), which lists every [background job](/reference/jobs) **checkship** queues.

::: info
Some jobs can be [cancelled](/reference/jobs#cancellation), cleanups included, because in rare cases they fail: the target may no longer be reachable, for example. **checkship** cleans up whatever it deployed, so cancelling the cleanup is what lets the application go in that case.
:::

After a few seconds, the application is gone.

Do the same for the target: open **the target page**, go to its **Settings** tab and click **Delete target** in **Danger zone**, then confirm. While an application still deploys to the target, **Danger zone** names it, and the target can't be deleted yet.

And we're done: everything has been cleaned up.
