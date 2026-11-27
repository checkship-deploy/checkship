# Backups

**checkship** backs up the named volumes of an application environment and puts them back, in the same environment, in another environment or in another application. Backups are also taken on their own before **checkship** deletes data: when you delete an application or move an environment to another target.

Each backup is one `.tar.gz` archive. It is always written to **checkship**'s data folder first, then copied to the [storage](#storage) you choose: the built-in S3 server **checkship** runs for you, an S3-compatible server you run, or nothing else.

## What is backed up {#what}

Every **named volume** declared in your compose file, for the environment you back up:

```yml
services:
  db:
    image: postgres:17
    volumes:
      - dbdata:/var/lib/postgresql/data # backed up
      - ./init:/docker-entrypoint-initdb.d # bind mount: not backed up
volumes:
  dbdata:
```

The **Backups** tab of an application lists the volumes of each environment, read live from its target, and the mounts a backup leaves out, with the reason:

| Mount                              | Backed up | Why                                                                               |
| ---------------------------------- | --------- | --------------------------------------------------------------------------------- |
| Named volume (`dbdata:/data`)      | Yes       | Created by compose for this environment, labelled by **checkship**                |
| Bind mount (`./cache:/cache`)      | No        | A folder on the server, not a Docker volume. Declare a named volume to include it |
| Anonymous volume (`- /data`)       | No        | It has no name to restore it under. Give it a name in the compose file            |
| External volume (`external: true`) | No        | Managed outside **checkship**                                                     |

::: tip
Named volumes are also the only data **checkship** keeps when you redeploy, so moving bind mounts to named volumes is worth it even without backups.
:::

## Taking a backup {#taking}

From the **Backups** tab of an application, **Back up now** opens a drawer:

- **Environment**: production or staging.
- **Volumes**: every named volume is ticked, each saying when it was last in a good backup and its size then. Untick the ones you don't need.
- **Stop containers while copying**: on by default. The containers of the environment stop until the copy ends, then start again, so a database is copied in a consistent state. The environment is unavailable meanwhile. Turned off, containers keep running and a file that changes during the copy may be saved half-written, which can corrupt a database.

The backup shows in the list as **Running**, then **Succeeded** with its size and where it is stored. Its details show its steps: requested, copied (how long it took and its size), then stored on the storage or kept only on its target, with what it holds per volume and every restore made from it.

### How the copy works {#how}

The copy happens on the target, through the same Docker connection deployments use (`ssh://` for a remote target):

1. If asked, the running containers of the environment are stopped.
2. For each volume, a helper container (`busybox:1.37` by default, see [settings](#settings)) is **created but never started**, with the volume mounted read-only. Docker streams the volume's files out of it as a tar.
3. The files go into the archive on **checkship**'s server, then the helper is removed.
4. The containers stopped in step 1 are started again, even when the copy failed.

Nothing is written on the target's disk and no network is given to the helper. The helper image is pulled on a target the first time it backs up or restores.

::: warning Databases
Stopping containers gives a consistent copy of the files. If a database must stay up, or you need point-in-time recovery, also use its own tool (`pg_dump`, `mysqldump`, `mongodump`) on a schedule. A file copy of a running database is not guaranteed to restore.
:::

## Freshness {#freshness}

The **Backups** page starts with its figures, then **Freshness**: one row per application environment with how old its last good backup is, its size and whether a copy is off the server (on the storage) or only on its target. The age of the newest backup you could restore is what tells you how much you would lose today, so the rows needing attention come first:

| Tone      | Last good backup                                                   |
| --------- | ------------------------------------------------------------------ |
| Green     | Less than a day ago                                                |
| No colour | Less than a week ago                                               |
| Amber     | Older than a week                                                  |
| Red       | Never: a deployed environment with named volumes and no backup yet |

**checkship** has no backup schedule yet, so the tones are fixed. A row opens the application's **Backups** tab, where each volume says the same about itself.

## Where backups go {#storage}

Choose it on **Backups › Storage**. Whatever you choose, the archive is first written to the data folder, and every backup records where its copies are:

| Where it is stored   | Meaning (on each row's details line)                                                         |
| -------------------- | -------------------------------------------------------------------------------------------- |
| Here + storage       | In the data folder and in the storage                                                        |
| Here                 | In the data folder only (no storage set)                                                     |
| Storage only         | Uploaded, and the local file removed (see [keep a local copy](#keep-local))                  |
| Here · upload failed | The storage was unreachable: the badge reads **Saved here only**, see [below](#storage-down) |

### checkship's data folder {#data-folder}

Archives live in `<data>/backups/<application id>/<backup id>.tar.gz` (`<data>` is [`DATA_PATH`](/guide/configuration), `/checkship/data` in the official image). They are readable by **checkship**'s user only (mode `0600`). An archive is written to `<data>/backups/.partial/` first and moved into place once complete, so a crash never leaves a half archive behind.

This is enough to undo mistakes, but if the server is lost its backups are lost with it. Pick one of the storages below to keep a copy elsewhere.

### Built-in storage {#builtin}

**checkship** runs an S3 server for you on one of your targets: [versitygw](https://github.com/versity/versitygw) (Versity S3 Gateway, Apache-2.0), with its POSIX backend. Every object it stores is a plain file in a folder, so archives stay readable even without the gateway.

It is set up automatically the **first time checkship starts** when its Docker engine is available. If there is no local target yet, checkship creates one called **This machine** and runs the storage there. This does not depend on `EXPOSED_ON`; that setting is only needed if you also want checkship exposed through its proxy. If Docker is unavailable, checkship remains usable and backups stay in the data folder until storage can be configured.

What **checkship** does on the chosen target:

- pulls and starts the separate VersityGW container in the compose project `checkship-internal-storage-<target id>` with the image `versity/versitygw:v1.11.0` by default (see [settings](#settings));
- keeps its data in the volume `checkship-internal-storage-<target id>_data`, mounted at `/data`, with one folder per bucket: `checkship-backups/<application id>/<backup id>.tar.gz`;
- generates an access key and a secret key, kept in **checkship**'s database and passed to the gateway as `ROOT_ACCESS_KEY` and `ROOT_SECRET_KEY`;
- creates the bucket `checkship-backups`, then writes, reads and deletes a small test object before showing the storage as **Ready**.

To reach it, **checkship** tries these addresses in order and keeps the first that answers:

| Address                  | When it works                                                                                               |
| ------------------------ | ----------------------------------------------------------------------------------------------------------- |
| `checkship-storage:7070` | **checkship** runs in a container on the same Docker engine; checkship joins the storage network           |
| `s3.<target host>`       | The target has a URL: its proxy routes `s3.` of its domain to the storage, over HTTPS if the target uses it |
| `127.0.0.1:<port>`       | A local target, with **checkship** running directly on the Docker host; the port may move if occupied        |
| `<target host>:<port>`   | A remote target without a URL; its Docker host publishes the gateway port for checkship to reach             |

The storage page starts with four figures: where copies go, its state (**Starting**, **Ready** or **Failed** with the reason), when the last copy left the server, and how many good backups are still only on their target. Below, the settings show the address in use, the bucket and the access key. A failed setup can be started again by saving the page.

::: tip Put it on another server
On the server **checkship** runs on, the built-in storage protects your applications deployed elsewhere, but not **checkship** itself. Choose another target in **Runs on** so losing one server never loses both. When you move it, archives are copied to the new target first, each one checked for size, and the old storage is removed only once every archive arrived. The keys stay the same.
:::

A target that runs the built-in storage cannot be deleted (`target_hosts_backup_storage`). Move the storage first.

#### Reading archives without checkship {#builtin-recovery}

Since objects are plain files, you can copy them straight out of the volume:

```sh
docker run --rm -v checkship-internal-storage-<target id>_data:/data:ro -v "$PWD":/out busybox:1.37 \
  cp -r /data/checkship-backups /out/
```

### Your own S3-compatible storage {#external}

Any server that speaks the S3 API works. **checkship** talks to it with [minio-go](https://github.com/minio/minio-go), which sends only the checksum headers a server asks for, so self-hosted servers that reject the newer AWS SDK defaults work too.

| Field                  | What to put                                                                                                     |
| ---------------------- | --------------------------------------------------------------------------------------------------------------- |
| Endpoint               | Host and port, without scheme or bucket: `garage.example.com:3900`. A pasted URL loses its scheme and path      |
| Region                 | The server's region. Garage uses `garage` unless you set `s3_region`, most others accept `us-east-1`            |
| Bucket                 | Created if it does not exist                                                                                    |
| Folder prefix          | Optional. Archives go under `<prefix>/<application id>/<backup id>.tar.gz`, so several tools can share a bucket |
| Access key, secret key | A key allowed to read, write, list and delete in the bucket. Leave the secret empty to keep the saved one       |
| Use HTTPS              | Off only for a server on a private network                                                                      |
| Path-style addressing  | `host/bucket/key` instead of `bucket.host/key`. Needed by most self-hosted servers                              |

The presets fill the endpoint and region for these servers:

| Server        | License    | Example endpoint                     | Region            | Notes                                                              |
| ------------- | ---------- | ------------------------------------ | ----------------- | ------------------------------------------------------------------ |
| Garage        | AGPL-3.0   | `garage.example.com:3900`            | `garage`          | Light, built for small clusters. [Setup below](#garage)            |
| SeaweedFS     | Apache-2.0 | `seaweed.example.com:8333`           | `us-east-1`       | [Setup below](#seaweedfs)                                          |
| RustFS        | Apache-2.0 | `rustfs.example.com:9000`            | `us-east-1`       | Use 1.0.0 or later: earlier releases had critical credential flaws |
| Ceph RGW      | LGPL-2.1   | `rgw.example.com:7480`               | zonegroup name    |                                                                    |
| AWS S3        | -          | `s3.eu-west-3.amazonaws.com`         | bucket region     | Path-style optional                                                |
| Cloudflare R2 | -          | `<account>.r2.cloudflarestorage.com` | `auto`            |                                                                    |
| Backblaze B2  | -          | `s3.eu-central-003.backblazeb2.com`  | from the endpoint |                                                                    |

**Test connection** checks the settings without saving them: it creates the bucket if needed, writes a small object, reads it back and deletes it. **Save** runs the same check and refuses settings that fail it (`storage_unreachable`).

::: info Credentials
Keys are kept in **checkship**'s database and masked in the API (`****`), like [registry](/reference/registries) passwords. They are not encrypted at rest: protect the data folder as you would the server itself.
:::

#### Garage {#garage}

Deploy [Garage](https://garagehq.deuxfleurs.fr) as an application, on a target other than your apps' if you can. Its configuration goes in through compose `configs`, and its S3 port is published:

```yml
services:
  garage:
    image: dxflrs/garage:v2.5.0
    restart: unless-stopped
    ports:
      - "3900:3900"
    configs:
      - source: garage_toml
        target: /etc/garage.toml
    volumes:
      - meta:/var/lib/garage/meta
      - data:/var/lib/garage/data
configs:
  garage_toml:
    content: |
      metadata_dir = "/var/lib/garage/meta"
      data_dir = "/var/lib/garage/data"
      db_engine = "sqlite"
      replication_factor = 1
      rpc_bind_addr = "[::]:3901"
      rpc_public_addr = "127.0.0.1:3901"
      rpc_secret = "<openssl rand -hex 32>"
      [s3_api]
      s3_region = "garage"
      api_bind_addr = "[::]:3900"
      root_domain = ".s3.garage.localhost"
      [admin]
      api_bind_addr = "[::]:3903"
      admin_token = "<openssl rand -hex 16>"
volumes:
  meta:
  data:
```

Once it runs, give the node a role, then create a key and the bucket, with Garage's own CLI inside the container:

```sh
G="docker exec <garage container> /garage"
$G layout assign -z dc1 -c 10G "$($G node id -q | cut -d@ -f1)"
$G layout apply --version 1
$G key create checkship          # prints the Key ID and Secret key
$G bucket create checkship-backups
$G bucket allow --read --write --owner checkship-backups --key checkship
```

On **Backups › Storage**, pick **Your own S3-compatible storage**, the **Garage** preset, then the endpoint (`<server>:3900`), the key and its secret. Path-style stays on.

#### SeaweedFS {#seaweedfs}

[SeaweedFS](https://github.com/seaweedfs/seaweedfs) runs its master, volume server, filer and S3 gateway in one container with `weed server`. The S3 identities go in a JSON file:

```yml
services:
  seaweed:
    image: chrislusf/seaweedfs:4.53
    restart: unless-stopped
    command:
      [
        "server",
        "-dir=/data",
        "-ip.bind=0.0.0.0",
        "-master.volumeSizeLimitMB=1024",
        "-volume.max=20",
        "-s3",
        "-s3.port=8333",
        "-s3.config=/etc/seaweedfs/s3.json",
      ]
    ports:
      - "8333:8333"
    configs:
      - source: s3_json
        target: /etc/seaweedfs/s3.json
    volumes:
      - data:/data
configs:
  s3_json:
    content: |
      {"identities": [{"name": "checkship",
        "credentials": [{"accessKey": "<access key>", "secretKey": "<secret key>"}],
        "actions": ["Admin", "Read", "Write", "List", "Tagging"]}]}
volumes:
  data:
```

On **Backups › Storage**, pick the **SeaweedFS** preset, then the endpoint (`<server>:8333`) and the keys from the JSON file. The bucket is created by **checkship**.

::: tip Small servers
SeaweedFS reserves its volumes up front (`-master.volumeSizeLimitMB` × `-volume.max`). Lower both on a small disk.
:::

### checkship's data folder only {#local-only}

No copy anywhere else. Choose it to stop copying to a storage. Archives already in the storage stay there.

### Keep a local copy {#keep-local}

With a storage set, **Keep a copy in checkship's data folder** is on by default: downloads and restores still work when the storage is down, at the cost of disk space on **checkship**'s server. Turned off, the local file is removed once the upload succeeds, and downloads and restores read from the storage.

**checkship** never removes the last copy of a backup: if the upload fails, the local file stays.

### When the storage is down {#storage-down}

A backup is never lost because the storage is unreachable. The archive stays in the data folder, the backup succeeds with the badge **Saved here only**, and its row reads **Here · upload failed**.

Once the storage is back, **Upload again** in the backup's menu copies it there (a background job, `deployment.command.store_backup`).

## Restoring {#restoring}

**Restore…** in a backup's menu, or in its details, opens a drawer:

- **Into application** and **Environment**: where the volumes go. It defaults to where the backup came from, but any application that is not being deleted works. Volumes are matched by their **name in the compose file**, so a production backup restores into staging, or into a new application declaring the same volume names.
- **Volumes**: the volumes of the archive, each saying whether it replaces an existing volume or will be created by the restore.
- **Back up the current data first**: on by default. The volumes about to be replaced are backed up first (trigger **Before restore**), so the restore can be undone with that backup.

Before you confirm, the drawer lists **What will happen**, in the order below, following the volumes you tick; with **Back up the current data first** off, it warns that there is no way back. A dialog then names the volumes it replaces. When you confirm, a background job (`deployment.command.restore_volumes`):

1. waits for any deployment of the destination environment to finish;
2. reads the whole archive and compares its SHA-256 with the one recorded when the backup was made: a damaged archive is refused (`checksum_mismatch`) before anything changes;
3. takes the **Before restore** backup if asked, and stops if that backup fails;
4. stops the containers that use the volumes, empties each volume, gives its root folder back the owner and mode it had, copies the files back (owners included), then starts the containers again.

A volume the destination does not declare yet is created the way compose would, with the same labels, so the next deployment adopts it.

Restores are recorded and listed in the backup's details with their outcome. A failed restore is **never retried on its own**, since each attempt stops the application: fix the cause and restore again.

::: warning
Containers stop while their volumes are replaced: the environment is unavailable during the restore. There is no way to restore into a running container, since its files would change under it.
:::

## Downloading and uploading {#files}

**Download** streams the archive from the data folder, or from the storage when there is no local copy. The file is named `<application>-<environment>-<date>.tar.gz` and can be opened with any `tar`:

```sh
tar -tzf blog-production-2026-11-15-1402.tar.gz
# manifest.json
# volumes/dbdata/
# volumes/dbdata/PG_VERSION
# ...
```

**Upload backup** adds an archive made by **checkship**, on this server or another one, to an application. The server checks that:

- the file is under the upload limit (`BACKUPS_MAX_UPLOAD_SIZE`, 5gb by default): `archive_too_large`;
- it is a gzip file whose first entry is a `manifest.json` in a format it knows: `manifest_missing`, `unsupported_archive_format`;
- every entry stays inside its volume: no absolute path, no `..`, no link pointing out of the volume: `invalid_archive_entry`.

The uploaded archive becomes a backup with the trigger **Uploaded**, ready to restore.

### Archive format {#format}

```txt
manifest.json            first entry: format version, checkship version, application id and name,
                         environment, target, volumes (compose name, Docker name, driver),
                         mounts left out
volumes/<name>/          the root of each volume, with its owner and mode
volumes/<name>/...       its files, paths relative to the volume
```

Volumes appear one after the other, in the order of the manifest. Owners, modes, symbolic links and hard links inside a volume are kept.

## Backups taken on their own {#safety-nets}

### Before deleting an application {#before-delete}

The deletion dialog has **Back up volumes first**, on by default. Each environment's volumes are backed up before they are removed (trigger **Before delete**). If a backup fails, the cleanup job fails and is [retried](/reference/jobs), and **nothing is deleted** until it succeeds, or until you [cancel the cleanup](/reference/jobs#cancellation).

The backups stay under **Backups** after the application is gone, shown as `<name> (deleted)`, and can be restored into another application. From the API: `DELETE /api/v1/apps/:id?backup_volumes=false` skips them.

### When an environment changes target {#move}

Changing the target of an environment in an application's settings shows **Move volume data to the new target**, on by default:

1. the environment is redeployed on the new target, with empty volumes;
2. the volumes on the old target are backed up (trigger **Before target change**);
3. that backup is restored on the new target, after the redeploy: containers stop, the volumes are filled, they start again;
4. the containers and volumes on the old target are removed.

Anything the application wrote on the new target between steps 1 and 3 is replaced. Turned off, the new target starts with empty volumes, and the old volumes are still backed up before removal.

Both are skipped when `BACKUPS_BEFORE_CLEANUP` is `false`, except a move you asked for.

## States {#states}

| Badge           | Meaning                                                                      |
| --------------- | ---------------------------------------------------------------------------- |
| Waiting         | Queued, for example behind a deployment of the same environment              |
| Running         | Copying. The row reads **Copying…**                                          |
| Succeeded       | Done, with its size and where it is stored                                   |
| Saved here only | Done, but the upload to the storage failed: **Upload again** once it is back |
| Failed          | Nothing saved. The row shows the error code; **Retry** runs it again         |

Backups wait for any running deployment of the same environment and run one at a time on their own [runner](/guide/configuration#runners), so a long copy never holds back deployments or cleanups.

## Settings {#settings}

| yaml path / env name                               | Description                                                                                  | Default                     |
| -------------------------------------------------- | -------------------------------------------------------------------------------------------- | --------------------------- |
| backups.helper_image<br>BACKUPS_HELPER_IMAGE       | Image that mounts each volume during a copy. Any image with `sh` and `find` works            | `busybox:1.37`              |
| backups.storage_image<br>BACKUPS_STORAGE_IMAGE     | Image of the built-in storage                                                                | `versity/versitygw:v1.11.0` |
| backups.max_upload_size<br>BACKUPS_MAX_UPLOAD_SIZE | Largest archive **Upload backup** accepts, with a unit: b, kb, mb, gb or tb (powers of 1024) | `5gb`                       |
| backups.before_cleanup<br>BACKUPS_BEFORE_CLEANUP   | Back up volumes before **checkship** removes them (application deleted, environment moved)   | `true`                      |

The storage itself (mode, target, endpoint, keys) is set on **Backups › Storage**, which also lists the two images in use.

## Errors {#errors}

| Code                          | Meaning and what to do                                                                                                                      |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `no_volumes`                  | The environment declares no named volumes, or was never deployed                                                                            |
| `volume_not_found`            | A volume asked for is not in the environment or the archive                                                                                 |
| `helper_image_pull_failed`    | The target could not pull the helper image: check its network and Docker Hub limits, or set `BACKUPS_HELPER_IMAGE` to an image it can reach |
| `volume_copy_failed`          | Docker refused a copy: the target's log has the reason (disk full, volume driver error)                                                     |
| `target_connect_failed`       | The target could not be reached                                                                                                             |
| `storage_unreachable`         | The storage refused the connection or the keys: check the endpoint, HTTPS, path-style, keys and bucket                                      |
| `storage_not_ready`           | The built-in storage is still starting, or failed: see **Backups › Storage**                                                                |
| `storage_move_failed`         | Moving the built-in storage to another target could not copy every archive. The old storage is kept                                         |
| `checksum_mismatch`           | The archive changed since the backup was made. It is not restored                                                                           |
| `manifest_missing`            | Not a **checkship** backup                                                                                                                  |
| `archive_too_large`           | Over `BACKUPS_MAX_UPLOAD_SIZE`                                                                                                              |
| `invalid_archive_entry`       | An entry would land outside its volume                                                                                                      |
| `backup_not_finished`         | The backup is still running: wait before deleting it                                                                                        |
| `target_hosts_backup_storage` | The target runs the built-in storage: move it before deleting the target                                                                    |

## API {#api}

These routes accept the [API key](/reference/api) as well as a session:

```http
# Volumes of an environment and the mounts left out
GET /api/v1/apps/:id/volumes?environment=production
# Backups of an application, newest first
GET /api/v1/apps/:id/backups
# Back up an environment ({"environment", "volumes": [] for all, "stop_containers"})
POST /api/v1/apps/:id/backups
# Add an archive (multipart: archive, optional environment)
POST /api/v1/apps/:id/backups/upload
# Every backup, including deleted applications
GET /api/v1/backups
# One backup with its restores
GET /api/v1/backups/:id
GET /api/v1/backups/:id/download
# Restore ({"app_id", "environment", "volumes", "snapshot_first"}, all optional)
POST /api/v1/backups/:id/restore
# Copy to the storage again, run a failed backup again
POST /api/v1/backups/:id/store
POST /api/v1/backups/:id/retry
DELETE /api/v1/backups/:id
```

The storage settings use the dashboard session only: `GET /api/v1/backup-storage`, `PUT /api/v1/backup-storage` and `POST /api/v1/backup-storage/test`.

```sh
# Back up production from CI before a risky deployment
curl -X POST -H "Authorization: Bearer <API key>" -H "Content-Type: application/json" \
  -d '{"environment": "production", "volumes": [], "stop_containers": true}' \
  https://checkship.example.com/api/v1/apps/<app id>/backups
```
