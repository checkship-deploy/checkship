package startup

import (
	"context"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_backup_storage"

	"github.com/checkship-deploy/checkship/internal/auth/app/create_first_account"
	"github.com/checkship-deploy/checkship/internal/auth/domain"
	authinfra "github.com/checkship-deploy/checkship/internal/auth/infra"
	"github.com/checkship-deploy/checkship/internal/deployment/app/expose_checkship_container"
	deploymentdomain "github.com/checkship-deploy/checkship/internal/deployment/domain"
	deploymentinfra "github.com/checkship-deploy/checkship/internal/deployment/infra"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/memory"
	bussqlite "github.com/checkship-deploy/checkship/pkg/bus/sqlite"
	"github.com/checkship-deploy/checkship/pkg/live"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
)

type (
	// Represents a services root containing every services used by a server.
	ServerRoot interface {
		Cleanup() error
		Bus() bus.Dispatcher
		Logger() log.Logger
		UsersReader() domain.UsersReader
		ScheduledJobsStore() bus.ScheduledJobsStore
		Live() *live.Hub
		LogSignals() *live.Signals
	}

	ServerOptions interface {
		deploymentinfra.Options

		AppExposedUrl() monad.Maybe[deploymentdomain.Url]
		DefaultEmail() string
		DefaultPassword() string
		RunnersDefinitions([]string, log.Logger) ([]bus.RunnerDefinition, error)
		ConnectionString() string
	}

	serverRoot struct {
		bus            bus.Bus
		logger         log.Logger
		db             *sqlite.Database
		usersReader    domain.UsersReader
		schedulerStore bus.ScheduledJobsStore
		scheduler      bus.RunnableScheduler
		live           *live.Hub
		logSignals     *live.Signals
	}
)

// Instantiate a new server root, registering and initializing every services
// needed by the server.
func Server(options ServerOptions, logger log.Logger) (ServerRoot, error) {
	s := &serverRoot{
		logger: logger,
	}

	s.bus = memory.NewBus()

	db, err := sqlite.Open(options.ConnectionString(), s.logger, s.bus)

	if err != nil {
		return nil, err
	}

	s.db = db

	s.schedulerStore = bussqlite.NewScheduledJobsStore(s.db)

	if err = s.schedulerStore.Setup(); err != nil {
		return nil, err
	}

	// Runners are started at the end, once every job handler is registered
	s.scheduler = bus.NewScheduler(s.schedulerStore, s.logger, s.bus)

	// Setup auth infrastructure
	if s.usersReader, err = authinfra.Setup(s.logger, s.db, s.bus); err != nil {
		return nil, err
	}

	// Wakes the live deployment log streams up each time a log is written
	s.logSignals = live.NewSignals()

	// Setups deployment infrastructure
	if err = deploymentinfra.Setup(
		options,
		s.logger,
		s.db,
		s.bus,
		s.scheduler,
		s.logSignals,
	); err != nil {
		return nil, err
	}

	// Live hints for the dashboard (server-sent events): 5 streams per user, 100 in all, 32 buffered
	// hints per stream. Registered before anything runs, the bus handlers map isn't synchronized.
	s.live = live.NewHub(5, 100, 32)
	deploymentinfra.RegisterLiveHints(s.bus, s.live.Publish)

	// Create the first account if needed
	uid, err := bus.Send(s.bus, context.Background(), create_first_account.Command{
		Email:    options.DefaultEmail(),
		Password: options.DefaultPassword(),
	})

	if err != nil {
		return nil, err
	}

	// Create the target needed to expose checkship itself and manage certificates if needed
	if exposedUrl, isSet := options.AppExposedUrl().TryGet(); isSet {
		container := exposedUrl.User().Get("")

		s.logger.Infow("exposing checkship container using the local target, creating it if needed, the container may restart once done",
			"container", container)

		if _, err := bus.Send(s.bus, domain.WithUserID(context.Background(), domain.UserID(uid)), expose_checkship_container.Command{
			Container: container,
			Url:       exposedUrl.WithoutUser().String(),
		}); err != nil {
			return nil, err
		}
	}

	// Install built-in backup storage on the local target, creating the target if needed.
	// Saved built-in configurations are also requeued to restore their Docker network access.
	if _, err := bus.Send(s.bus, domain.WithUserID(context.Background(), domain.UserID(uid)), update_backup_storage.EnsureCommand{}); err != nil {
		s.logger.Warnw("could not set up the built-in backup storage, backups stay in the data folder", "error", err)
	}

	// Every job type is known once the handlers are registered (bus.Register fills
	// bus.Marshallable), so the runners configuration can be checked against them.
	runners, err := options.RunnersDefinitions(bus.Marshallable.Keys(), s.logger)

	if err != nil {
		return nil, err
	}

	s.scheduler.Start(runners...)

	return s, nil
}

func (s *serverRoot) Cleanup() error {
	s.logger.Debug("cleaning server services")

	s.scheduler.Stop()

	return s.db.Close()
}

func (s *serverRoot) Bus() bus.Dispatcher                        { return s.bus }
func (s *serverRoot) Logger() log.Logger                         { return s.logger }
func (s *serverRoot) UsersReader() domain.UsersReader            { return s.usersReader }
func (s *serverRoot) ScheduledJobsStore() bus.ScheduledJobsStore { return s.schedulerStore }
func (s *serverRoot) Live() *live.Hub                            { return s.live }
func (s *serverRoot) LogSignals() *live.Signals                  { return s.logSignals }
