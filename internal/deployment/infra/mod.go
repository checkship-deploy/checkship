package infra

import (
	"context"
	"errors"
	"github.com/checkship-deploy/checkship/internal/deployment/app/backup_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/app/configure_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/app/delete_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_backup_archive"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_backup_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/app/request_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/request_restore"
	"github.com/checkship-deploy/checkship/internal/deployment/app/restore_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/app/retry_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/store_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_backup_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/app/upload_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/backupstore"

	"github.com/checkship-deploy/checkship/internal/deployment/app/cancel_deployment"
	"github.com/checkship-deploy/checkship/internal/deployment/app/check_app_domains"
	"github.com/checkship-deploy/checkship/internal/deployment/app/cleanup_app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/cleanup_target"
	"github.com/checkship-deploy/checkship/internal/deployment/app/cloudflare_setup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/configure_target"
	"github.com/checkship-deploy/checkship/internal/deployment/app/create_app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/create_registry"
	"github.com/checkship-deploy/checkship/internal/deployment/app/create_target"
	"github.com/checkship-deploy/checkship/internal/deployment/app/delete_app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/delete_registry"
	"github.com/checkship-deploy/checkship/internal/deployment/app/delete_target"
	"github.com/checkship-deploy/checkship/internal/deployment/app/deploy"
	"github.com/checkship-deploy/checkship/internal/deployment/app/expose_checkship_container"
	"github.com/checkship-deploy/checkship/internal/deployment/app/fail_pending_deployments"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_logs"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_services"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment_log"
	"github.com/checkship-deploy/checkship/internal/deployment/app/promote"
	"github.com/checkship-deploy/checkship/internal/deployment/app/queue_deployment"
	"github.com/checkship-deploy/checkship/internal/deployment/app/reconfigure_target"
	"github.com/checkship-deploy/checkship/internal/deployment/app/redeploy"
	"github.com/checkship-deploy/checkship/internal/deployment/app/request_app_cleanup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/request_target_cleanup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_registry"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_target"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/artifact"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/cloudflare"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/domaincheck"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/provider"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/provider/docker"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/running"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source/archive"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source/git"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source/raw"
	deploymentsqlite "github.com/checkship-deploy/checkship/internal/deployment/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/live"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
)

type Options interface {
	artifact.LocalOptions
	backup_volumes.Options
	upload_backup.Options
	cleanup_app.Options
	configure_storage.Options
	archive.Options

	// The address checkship itself is exposed on (EXPOSED_ON), if any: no application can
	// take it as a custom domain.
	AppExposedUrl() monad.Maybe[domain.Url]
}

// Setup the deployment module and register everything needed in the given
// bus.
func Setup(
	opts Options,
	logger log.Logger,
	db *sqlite.Database,
	b bus.Bus,
	scheduler bus.Scheduler,
	logSignals *live.Signals, // Notified with the log path each time a deployment log is written
) error {
	appsStore := deploymentsqlite.NewAppsStore(db)
	deploymentsStore := deploymentsqlite.NewDeploymentsStore(db)
	targetsStore := deploymentsqlite.NewTargetsStore(db)
	registriesStore := deploymentsqlite.NewRegistriesStore(db)
	backupsStore := deploymentsqlite.NewBackupsStore(db)
	restoresStore := deploymentsqlite.NewRestoresStore(db)
	backupStorageStore := deploymentsqlite.NewBackupStorageStore(db)
	deploymentQueryHandler := deploymentsqlite.NewGateway(db)

	artifactManager := artifact.NewLocal(opts, logger, artifact.WithLogWritten(logSignals.Notify))
	runningDeployments := running.New()

	sourceFacade := source.NewFacade(
		raw.New(),
		archive.New(opts),
		git.New(appsStore),
	)

	dock := docker.New(logger)
	providerFacade := provider.NewFacade(
		dock,
	)

	backupFiles := backupstore.New(opts, backupstore.Resolver(backupStorageStore))
	storageChecker := backupstore.NewChecker()
	backupRunner := backup_volumes.NewRunner(targetsStore, backupsStore, backupsStore, providerFacade, backupFiles, opts, logger)

	bus.Register(b, expose_checkship_container.Handler(targetsStore, targetsStore, dock))
	domainsRules := create_app.DomainsRules{Targets: targetsStore, Apps: appsStore, Reserved: opts.AppExposedUrl()}
	bus.Register(b, create_app.Handler(appsStore, appsStore, domainsRules))
	bus.Register(b, update_app.Handler(appsStore, appsStore, domainsRules))
	bus.Register(b, check_app_domains.Handler(appsStore, targetsStore, domaincheck.New()))

	// Connect Cloudflare
	cloudflareDeps := cloudflare_setup.Deps{
		Connections: deploymentsqlite.NewCloudflareConnectionsStore(db),
		Cloudflare:  cloudflare.New(),
		Targets:     targetsStore,
		Apps:        appsStore,
		Connector:   dock,
	}
	bus.Register(b, cloudflare_setup.ConnectHandler(cloudflareDeps))
	bus.Register(b, cloudflare_setup.StatusHandler(cloudflareDeps))
	bus.Register(b, cloudflare_setup.SyncHandler(cloudflareDeps))
	bus.Register(b, cloudflare_setup.SyncJobHandler(cloudflareDeps))
	bus.Register(b, cloudflare_setup.DisconnectHandler(cloudflareDeps))
	bus.Register(b, queue_deployment.Handler(appsStore, deploymentsStore, deploymentsStore, sourceFacade))
	bus.Register(b, deploy.Handler(deploymentsStore, deploymentsStore, artifactManager, sourceFacade, providerFacade, targetsStore, registriesStore, runningDeployments))
	bus.Register(b, cancel_deployment.Handler(deploymentsStore, deploymentsStore, runningDeployments))
	bus.Register(b, request_app_cleanup.Handler(appsStore, appsStore))
	bus.Register(b, delete_app.Handler(appsStore, appsStore, artifactManager))
	bus.Register(b, cleanup_app.Handler(targetsStore, deploymentsStore, providerFacade, cleanup_app.Backups{
		Apps:     appsStore,
		Reader:   backupsStore,
		Restores: restoresStore,
		Runner:   backupRunner,
	}))
	bus.Register(b, get_deployment_log.Handler(deploymentsStore, artifactManager))
	bus.Register(b, get_app_logs.Handler(appsStore, targetsStore, providerFacade))
	bus.Register(b, get_app_services.Handler(appsStore, targetsStore, providerFacade))
	bus.Register(b, redeploy.Handler(appsStore, deploymentsStore, deploymentsStore))
	bus.Register(b, promote.Handler(appsStore, deploymentsStore, deploymentsStore))
	bus.Register(b, create_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, configure_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, reconfigure_target.Handler(targetsStore, targetsStore))
	bus.Register(b, update_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, request_target_cleanup.Handler(targetsStore, targetsStore, appsStore, backupStorageStore))
	bus.Register(b, cleanup_target.Handler(targetsStore, deploymentsStore, providerFacade))
	bus.Register(b, delete_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, create_registry.Handler(registriesStore, registriesStore))
	bus.Register(b, update_registry.Handler(registriesStore, registriesStore))
	bus.Register(b, delete_registry.Handler(registriesStore, registriesStore))
	bus.Register(b, request_backup.Handler(appsStore, backupsStore))
	bus.Register(b, backup_volumes.Handler(backupsStore, backupRunner))
	bus.Register(b, upload_backup.Handler(appsStore, backupsStore, backupFiles, opts))
	bus.Register(b, retry_backup.Handler(backupsStore, backupsStore))
	bus.Register(b, delete_backup.Handler(backupsStore, backupsStore, backupFiles))
	bus.Register(b, store_backup.RequestHandler(backupsStore, scheduler))
	bus.Register(b, store_backup.Handler(backupsStore, backupsStore, backupFiles, logger))
	bus.Register(b, request_restore.Handler(backupsStore, appsStore, restoresStore))
	bus.Register(b, restore_volumes.Handler(restoresStore, restoresStore, backupsStore, appsStore, targetsStore, backupFiles, providerFacade, backupRunner, opts, logger))
	bus.Register(b, update_backup_storage.Handler(backupStorageStore, backupStorageStore, targetsStore, storageChecker))
	bus.Register(b, update_backup_storage.TestHandler(storageChecker))
	bus.Register(b, update_backup_storage.EnsureHandler(backupStorageStore, backupStorageStore, targetsStore, targetsStore, dock, scheduler))
	bus.Register(b, configure_storage.Handler(backupStorageStore, backupStorageStore, targetsStore, providerFacade, storageChecker, backupstore.NewMover(), opts, logger))
	bus.Register(b, get_backup_storage.Handler(backupStorageStore, opts))
	bus.Register(b, get_app_volumes.Handler(appsStore, targetsStore, providerFacade))
	bus.Register(b, get_backup_archive.Handler(backupsStore, backupFiles))
	bus.Register(b, deploymentQueryHandler.GetBackups)
	bus.Register(b, deploymentQueryHandler.GetBackupByID)
	bus.Register(b, deploymentQueryHandler.GetAllApps)
	bus.Register(b, deploymentQueryHandler.GetAppByID)
	bus.Register(b, deploymentQueryHandler.GetAllDeploymentsByApp)
	bus.Register(b, deploymentQueryHandler.GetDeploymentByID)
	bus.Register(b, deploymentQueryHandler.GetAllTargets)
	bus.Register(b, deploymentQueryHandler.GetTargetByID)
	bus.Register(b, deploymentQueryHandler.GetRegistries)
	bus.Register(b, deploymentQueryHandler.GetRegistryByID)
	bus.Register(b, deploymentQueryHandler.GetDashboardSummary)
	bus.Register(b, deploymentQueryHandler.GetDashboardApps)
	bus.Register(b, deploymentQueryHandler.GetDashboardActivity)

	bus.On(b, deploy.OnDeploymentCreatedHandler(scheduler))
	bus.On(b, backup_volumes.OnBackupRequestedHandler(scheduler))
	bus.On(b, backup_volumes.OnBackupRetriedHandler(scheduler))
	bus.On(b, configure_storage.OnBackupStorageChangedHandler(scheduler))
	bus.On(b, request_restore.OnRestoreRequestedHandler(scheduler))
	bus.On(b, redeploy.OnAppEnvChangedHandler(appsStore, deploymentsStore, deploymentsStore))
	bus.On(b, delete_app.OnAppCleanupRequestedHandler(scheduler))
	bus.On(b, cleanup_app.OnAppEnvChangedHandler(scheduler, opts))
	bus.On(b, cleanup_app.OnAppCleanupRequestedHandler(scheduler, opts))
	bus.On(b, fail_pending_deployments.OnTargetDeleteRequestedHandler(deploymentsStore))
	bus.On(b, fail_pending_deployments.OnAppCleanupRequestedHandler(deploymentsStore))
	bus.On(b, fail_pending_deployments.OnAppEnvChangedHandler(deploymentsStore))
	bus.On(b, cleanup_target.OnTargetCleanupRequestedHandler(scheduler))
	bus.On(b, configure_target.OnTargetCreatedHandler(scheduler))
	bus.On(b, configure_target.OnTargetStateChangedHandler(scheduler))
	bus.On(b, configure_target.OnDeploymentStateChangedHandler(targetsStore, targetsStore))
	bus.On(b, configure_target.OnAppEnvChangedHandler(targetsStore, targetsStore))
	bus.On(b, configure_target.OnAppCleanupRequestedHandler(targetsStore, targetsStore))
	bus.On(b, delete_target.OnTargetCleanupRequestedHandler(scheduler))
	bus.On(b, cloudflare_setup.OnAppCreatedHandler(scheduler))
	bus.On(b, cloudflare_setup.OnAppEnvChangedHandler(scheduler))
	bus.On(b, cloudflare_setup.OnAppCleanupRequestedHandler(scheduler))
	bus.On(b, cloudflare_setup.OnTargetCleanupRequestedHandler(scheduler))

	if err := db.Migrate(deploymentsqlite.Migrations); err != nil {
		return err
	}

	if err := ReconfigureOutdatedProxies(context.Background(), targetsStore, logger); err != nil {
		return err
	}

	// Fail running deployments in case of a hard reset.
	return deploymentsStore.FailDeployments(context.Background(), errors.New("server_reset"), domain.FailCriteria{
		Status: monad.Value(domain.DeploymentStatusRunning),
	})
}
