package update_backup_storage_test

import (
	"context"
	"errors"
	"testing"

	authdomain "github.com/checkship-deploy/checkship/internal/auth/domain"
	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/configure_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_backup_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	deploymentsqlite "github.com/checkship-deploy/checkship/internal/deployment/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/must"
)

type localProvider struct{ err error }

func (p localProvider) PrepareLocal(context.Context) (domain.ProviderConfig, error) {
	return fixture.ProviderConfig(fixture.WithFingerprint("")), p.err
}

type queuedScheduler struct{ requests []bus.Schedulable }

func (s *queuedScheduler) Queue(_ context.Context, request bus.Schedulable, _ ...bus.JobOptions) error {
	s.requests = append(s.requests, request)
	return nil
}

func newHandler(t *testing.T, user authdomain.User, targetOptions ...fixture.SeedBuilder) (*fixture.Context, bus.RequestHandler[bus.UnitType, update_backup_storage.EnsureCommand], *queuedScheduler) {
	t.Helper()
	seeds := append([]fixture.SeedBuilder{fixture.WithUsers(&user)}, targetOptions...)
	ctx := fixture.PrepareDatabase(t, seeds...)
	scheduler := &queuedScheduler{}
	storage := deploymentsqlite.NewBackupStorageStore(ctx.DB)
	handler := update_backup_storage.EnsureHandler(storage, storage, ctx.TargetsStore, ctx.TargetsStore, localProvider{}, scheduler)
	return ctx, handler, scheduler
}

func Test_EnsureBackupStorage(t *testing.T) {
	t.Run("creates the local target and enables built-in storage on first run", func(t *testing.T) {
		user := authfixture.User()
		ctx, handler, scheduler := newHandler(t, user)
		_, err := handler(ctx.Context, update_backup_storage.EnsureCommand{})
		assert.Nil(t, err)

		settings := must.Panic(deploymentsqlite.NewBackupStorageStore(ctx.DB).Get(ctx.Context))
		assert.Equal(t, domain.BackupStorageBuiltin, settings.Mode())
		assert.Equal(t, domain.BackupStorageConfiguring, settings.Status())
		assert.True(t, settings.Saved())
		assert.Equal(t, domain.BuiltinStorageBucket, settings.Access().Bucket)
		assert.HasLength(t, 1, scheduler.requests)
		queued := assert.Is[configure_storage.Command](t, scheduler.requests[0])
		assert.Equal(t, settings.Version(), queued.Version)

		target, err := ctx.TargetsStore.GetLocalTarget(ctx.Context)
		assert.Nil(t, err)
		assert.Equal(t, target.ID(), settings.Target().MustGet())
	})

	t.Run("reuses an existing local target", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()), fixture.WithProviderConfig(fixture.ProviderConfig(fixture.WithFingerprint(""))))
		ctx, handler, _ := newHandler(t, user, fixture.WithTargets(&target))
		_, err := handler(ctx.Context, update_backup_storage.EnsureCommand{})
		assert.Nil(t, err)

		settings := must.Panic(deploymentsqlite.NewBackupStorageStore(ctx.DB).Get(ctx.Context))
		assert.Equal(t, target.ID(), settings.Target().MustGet())
	})

	t.Run("preserves explicit local and external choices", func(t *testing.T) {
		for _, mode := range []domain.BackupStorageMode{domain.BackupStorageLocal, domain.BackupStorageExternal} {
			t.Run(string(mode), func(t *testing.T) {
				user := authfixture.User()
				ctx, handler, scheduler := newHandler(t, user)
				store := deploymentsqlite.NewBackupStorageStore(ctx.DB)
				settings := must.Panic(store.Get(ctx.Context))
				if mode == domain.BackupStorageExternal {
					settings.UseExternal(domain.BackupStorageAccess{Endpoint: "s3.example.test", Bucket: "backups", AccessKey: "key", SecretKey: "secret"})
				} else {
					settings.UseDataFolderOnly()
				}
				assert.Nil(t, store.Write(ctx.Context, &settings))

				_, err := handler(ctx.Context, update_backup_storage.EnsureCommand{})
				assert.Nil(t, err)
				got := must.Panic(store.Get(ctx.Context))
				assert.Equal(t, mode, got.Mode())
				assert.HasLength(t, 0, scheduler.requests)
				_, err = ctx.TargetsStore.GetLocalTarget(ctx.Context)
				assert.True(t, errors.Is(err, apperr.ErrNotFound))
			})
		}
	})

	t.Run("rechecks saved built-in storage after restart", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()), fixture.WithProviderConfig(fixture.ProviderConfig(fixture.WithFingerprint(""))))
		ctx, handler, scheduler := newHandler(t, user, fixture.WithTargets(&target))
		store := deploymentsqlite.NewBackupStorageStore(ctx.DB)
		settings := must.Panic(store.Get(ctx.Context))
		settings.UseBuiltin(target.ID(), "access", "secret")
		settings.BuiltinConfigured(settings.Version(), "127.0.0.1:7070", false, nil)
		assert.Nil(t, store.Write(ctx.Context, &settings))

		_, err := handler(ctx.Context, update_backup_storage.EnsureCommand{})
		assert.Nil(t, err)
		assert.HasLength(t, 1, scheduler.requests)
		queued := assert.Is[configure_storage.Command](t, scheduler.requests[0])
		assert.Equal(t, settings.Version(), queued.Version)
		got := must.Panic(store.Get(ctx.Context))
		assert.Equal(t, domain.BackupStorageReady, got.Status())
	})

	t.Run("leaves default settings unchanged when local Docker preparation fails", func(t *testing.T) {
		user := authfixture.User()
		ctx := fixture.PrepareDatabase(t, fixture.WithUsers(&user))
		storage := deploymentsqlite.NewBackupStorageStore(ctx.DB)
		scheduler := &queuedScheduler{}
		dockerErr := errors.New("docker unavailable")
		handler := update_backup_storage.EnsureHandler(storage, storage, ctx.TargetsStore, ctx.TargetsStore, localProvider{err: dockerErr}, scheduler)
		_, err := handler(ctx.Context, update_backup_storage.EnsureCommand{})
		assert.ErrorIs(t, dockerErr, err)
		assert.HasLength(t, 0, scheduler.requests)
		settings := must.Panic(storage.Get(ctx.Context))
		assert.Equal(t, domain.BackupStorageLocal, settings.Mode())
	})
}
