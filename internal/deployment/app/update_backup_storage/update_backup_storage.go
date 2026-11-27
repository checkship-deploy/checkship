package update_backup_storage

import (
	"context"
	"errors"
	"time"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/configure_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/crypto"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/validate"
	"github.com/checkship-deploy/checkship/pkg/validate/strings"
)

type (
	// Choose where backups are copied besides the data folder.
	Command struct {
		bus.Command[bus.UnitType]

		Mode      string                    `json:"mode"`       // local, builtin or external
		TargetID  monad.Maybe[string]       `json:"target_id"`  // For builtin
		External  monad.Maybe[ExternalBody] `json:"external"`   // For external
		KeepLocal monad.Maybe[bool]         `json:"keep_local"` // Keep the copy in the data folder once stored
	}

	ExternalBody struct {
		Endpoint  string `json:"endpoint"`
		Region    string `json:"region"`
		Bucket    string `json:"bucket"`
		Prefix    string `json:"prefix"`
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"` // Empty keeps the current one
		UseSSL    bool   `json:"use_ssl"`
		PathStyle bool   `json:"path_style"`
	}

	// Check an S3-compatible storage can be used, without saving anything.
	TestCommand struct {
		bus.Command[bus.UnitType]

		External ExternalBody `json:"external"`
	}
)

func (Command) Name_() string     { return "deployment.command.update_backup_storage" }
func (TestCommand) Name_() string { return "deployment.command.test_backup_storage" }

func (b ExternalBody) access() domain.BackupStorageAccess {
	return domain.BackupStorageAccess{
		Endpoint:  b.Endpoint,
		Region:    b.Region,
		Bucket:    b.Bucket,
		Prefix:    b.Prefix,
		AccessKey: b.AccessKey,
		SecretKey: storage.SecretString(b.SecretKey),
		UseSSL:    b.UseSSL,
		PathStyle: b.PathStyle,
	}
}

func (b ExternalBody) validate(secretRequired bool) error {
	of := validate.Of{
		"external.endpoint":   validate.Field(b.Endpoint, strings.Required),
		"external.bucket":     validate.Field(b.Bucket, strings.Required),
		"external.access_key": validate.Field(b.AccessKey, strings.Required),
	}

	if secretRequired {
		of["external.secret_key"] = validate.Field(b.SecretKey, strings.Required)
	}

	return validate.Struct(of)
}

func Handler(
	reader domain.BackupStorageReader,
	writer domain.BackupStorageWriter,
	targets domain.TargetsReader,
	checker domain.BackupStorageChecker,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		settings, err := reader.Get(ctx)

		if err != nil {
			return bus.Unit, err
		}

		switch domain.BackupStorageMode(cmd.Mode) {
		case domain.BackupStorageLocal:
			if settings.Mode() != domain.BackupStorageLocal {
				settings.UseDataFolderOnly()
			}
		case domain.BackupStorageBuiltin:
			id, isSet := cmd.TargetID.TryGet()

			if !isSet {
				return bus.Unit, validate.NewError(validate.FieldErrors{"target_id": strings.ErrRequired})
			}

			target, err := targets.GetByID(ctx, domain.TargetID(id))

			if err != nil {
				return bus.Unit, err
			}

			current, isBuiltin := settings.Target().TryGet()

			// Set up again unless it already runs there and answers
			working := settings.Mode() == domain.BackupStorageBuiltin && isBuiltin && current == target.ID() &&
				settings.Status() != domain.BackupStorageFailed && settings.Access().Endpoint != ""

			if !working {
				accessKey, secretKey, err := Keys()

				if err != nil {
					return bus.Unit, err
				}

				settings.UseBuiltin(target.ID(), accessKey, secretKey)
			}
		case domain.BackupStorageExternal:
			body, isSet := cmd.External.TryGet()
			keepSecret := settings.Mode() == domain.BackupStorageExternal

			if !isSet {
				return bus.Unit, validate.NewError(validate.FieldErrors{"external": strings.ErrRequired})
			}

			if err = body.validate(!keepSecret); err != nil {
				return bus.Unit, err
			}

			access := body.access()

			if access.SecretKey == "" {
				access.SecretKey = settings.Access().SecretKey
			}

			if err = checker.Check(ctx, access); err != nil {
				return bus.Unit, apperr.Wrap(domain.ErrStorageUnreachable, err)
			}

			settings.UseExternal(access)
		default:
			return bus.Unit, validate.NewError(validate.FieldErrors{"mode": domain.ErrInvalidBackupStorageMode})
		}

		if keep, isSet := cmd.KeepLocal.TryGet(); isSet {
			settings.KeepLocalCopies(keep)
		}

		return bus.Unit, writer.Write(ctx, &settings)
	}
}

func TestHandler(checker domain.BackupStorageChecker) bus.RequestHandler[bus.UnitType, TestCommand] {
	return func(ctx context.Context, cmd TestCommand) (bus.UnitType, error) {
		if err := cmd.External.validate(true); err != nil {
			return bus.Unit, err
		}

		if err := checker.Check(ctx, cmd.External.access()); err != nil {
			return bus.Unit, apperr.Wrap(domain.ErrStorageUnreachable, err)
		}

		return bus.Unit, nil
	}
}

// Keys of a new built-in storage, generated once and kept on this server.
func Keys() (string, string, error) {
	access, err := crypto.RandomKey[string](20)

	if err != nil {
		return "", "", err
	}

	secret, err := crypto.RandomKey[string](40)

	return "ck" + access, secret, err
}

// Installs the built-in storage on the target checkship itself runs on, the
// first time checkship starts. Does nothing once settings were saved, or when
// there is no local target (backups then stay in the data folder).
type EnsureCommand struct {
	bus.Command[bus.UnitType]
}

func (EnsureCommand) Name_() string { return "deployment.command.ensure_backup_storage" }

func EnsureHandler(
	reader domain.BackupStorageReader,
	writer domain.BackupStorageWriter,
	targets domain.TargetsReader,
	targetWriter domain.TargetsWriter,
	localProvider LocalProvider,
	scheduler bus.Scheduler,
) bus.RequestHandler[bus.UnitType, EnsureCommand] {
	return func(ctx context.Context, _ EnsureCommand) (bus.UnitType, error) {
		settings, err := reader.Get(ctx)

		if err != nil {
			return bus.Unit, err
		}

		if settings.Saved() {
			if settings.Mode() != domain.BackupStorageBuiltin {
				return bus.Unit, nil // An explicit local or external choice always wins
			}

			// Re-run setup at each start: besides recovering a failed gateway, this
			// reconnects Checkship's container to the storage network after upgrades.
			return bus.Unit, queueSetup(ctx, scheduler, settings.Version())
		}

		local, err := targets.GetLocalTarget(ctx)

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				config, prepareErr := localProvider.PrepareLocal(ctx)

				if prepareErr != nil {
					return bus.Unit, prepareErr
				}

				requirement, prepareErr := targets.CheckConfigAvailability(ctx, config)

				if prepareErr != nil {
					return bus.Unit, prepareErr
				}

				local, prepareErr = domain.NewTarget(
					"This machine",
					requirement,
					auth.CurrentUser(ctx).MustGet(),
				)

				if prepareErr != nil {
					return bus.Unit, prepareErr
				}

				if prepareErr = targetWriter.Write(ctx, &local); prepareErr != nil {
					return bus.Unit, prepareErr
				}
			} else {
				return bus.Unit, err
			}
		}

		accessKey, secretKey, err := Keys()

		if err != nil {
			return bus.Unit, err
		}

		settings.UseBuiltin(local.ID(), accessKey, secretKey)

		if err = writer.Write(ctx, &settings); err != nil {
			return bus.Unit, err
		}

		// The storage-changed signal also queues this job in the application bus;
		// queue explicitly so setup remains guaranteed for other handler adapters.
		return bus.Unit, queueSetup(ctx, scheduler, settings.Version())
	}
}

func queueSetup(ctx context.Context, scheduler bus.Scheduler, version time.Time) error {
	return scheduler.Queue(ctx, configure_storage.Command{
		Version: version,
	}, bus.WithGroup("deployment.backup_storage"), bus.WithPolicy(bus.JobPolicyMerge))
}

// The Docker provider exposes its local configuration without requiring a
// remote host or credentials.
type LocalProvider interface {
	PrepareLocal(context.Context) (domain.ProviderConfig, error)
}
