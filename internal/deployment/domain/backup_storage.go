package domain

import (
	"context"
	"strings"
	"time"

	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

const (
	BackupStorageLocal    BackupStorageMode = "local"    // checkship's data folder only
	BackupStorageBuiltin  BackupStorageMode = "builtin"  // The S3 server checkship runs on a target
	BackupStorageExternal BackupStorageMode = "external" // An S3-compatible server the owner runs

	BackupStorageReady       BackupStorageStatus = "ready"
	BackupStorageConfiguring BackupStorageStatus = "configuring"
	BackupStorageFailed      BackupStorageStatus = "failed"

	BuiltinStorageBucket = "checkship-backups"
	BuiltinStorageRegion = "us-east-1" // versitygw accepts any region, this one is the S3 default
)

var (
	ErrInvalidBackupStorageMode = apperr.New("invalid_backup_storage_mode")
	ErrStorageNotReady          = apperr.New("storage_not_ready")
)

type (
	BackupStorageMode   string
	BackupStorageStatus string

	// How to reach an S3-compatible storage.
	BackupStorageAccess struct {
		Endpoint  string               `json:"endpoint"` // host:port, no scheme
		UseSSL    bool                 `json:"use_ssl"`
		PathStyle bool                 `json:"path_style"`
		Region    string               `json:"region"`
		Bucket    string               `json:"bucket"`
		Prefix    string               `json:"prefix"`
		AccessKey string               `json:"access_key"`
		SecretKey storage.SecretString `json:"secret_key"`
	}

	// Where backups are copied besides the data folder. There is only one for
	// the whole instance.
	BackupStorage struct {
		event.Emitter

		mode      BackupStorageMode
		target    monad.Maybe[TargetID] // Target running the built-in storage
		access    BackupStorageAccess
		keepLocal bool
		status    BackupStorageStatus
		errorCode monad.Maybe[string]
		version   time.Time // Changes every time the settings change, so an older configure job does nothing
	}

	BackupStorageReader interface {
		// Returns the settings, the default ones (data folder only) if never saved.
		Get(context.Context) (BackupStorage, error)
	}

	BackupStorageWriter interface {
		Write(context.Context, ...*BackupStorage) error
	}

	// Connects to a storage, creates its bucket if needed and checks an object can
	// be written, read and deleted.
	BackupStorageChecker interface {
		Check(context.Context, BackupStorageAccess) error
	}

	BackupStorageChanged struct {
		bus.Notification

		Mode      BackupStorageMode
		Target    monad.Maybe[TargetID]
		Access    BackupStorageAccess
		KeepLocal bool
		Status    BackupStorageStatus
		ErrorCode monad.Maybe[string]
		Version   time.Time
		Previous  monad.Maybe[TargetID] // Target that ran the built-in storage before this change
		// How to reach the built-in storage this change replaces, so its archives can be moved
		PreviousAccess monad.Maybe[BackupStorageAccess]
	}
)

func (BackupStorageChanged) Name_() string { return "deployment.event.backup_storage_changed" }

// Safe temporary settings before first-run setup selects the built-in storage.
func DefaultBackupStorage() BackupStorage {
	return BackupStorage{
		mode:      BackupStorageLocal,
		keepLocal: true,
		status:    BackupStorageReady,
	}
}

func BackupStorageFrom(scanner storage.Scanner) (s BackupStorage, err error) {
	var target monad.Maybe[string] // Maybe cannot scan into a named string type

	err = scanner.Scan(
		&s.mode,
		&target,
		&s.access.Endpoint,
		&s.access.UseSSL,
		&s.access.PathStyle,
		&s.access.Region,
		&s.access.Bucket,
		&s.access.Prefix,
		&s.access.AccessKey,
		&s.access.SecretKey,
		&s.keepLocal,
		&s.status,
		&s.errorCode,
		&s.version,
	)

	if id, isSet := target.TryGet(); isSet {
		s.target.Set(TargetID(id))
	}

	return s, err
}

// Keeps copies in the data folder only.
func (s *BackupStorage) UseDataFolderOnly() {
	s.change(BackupStorageLocal, monad.None[TargetID](), BackupStorageAccess{}, BackupStorageReady)
}

// Runs the built-in storage on a target. Keys are kept when it moves to another
// target, so existing access does not change. The endpoint is found once it runs.
func (s *BackupStorage) UseBuiltin(target TargetID, accessKey, secretKey string) {
	access := BackupStorageAccess{
		PathStyle: true,
		Region:    BuiltinStorageRegion,
		Bucket:    BuiltinStorageBucket,
		AccessKey: accessKey,
		SecretKey: storage.SecretString(secretKey),
	}

	if s.mode == BackupStorageBuiltin && s.access.AccessKey != "" {
		access.AccessKey = s.access.AccessKey
		access.SecretKey = s.access.SecretKey
	}

	s.change(BackupStorageBuiltin, monad.Value(target), access, BackupStorageConfiguring)
}

// Uses an S3-compatible storage the owner runs. A secret left empty keeps the
// current one, so the dashboard never has to send it back.
func (s *BackupStorage) UseExternal(access BackupStorageAccess) {
	access.Endpoint = normalizeEndpoint(access.Endpoint)

	if access.SecretKey == "" && s.mode == BackupStorageExternal {
		access.SecretKey = s.access.SecretKey
	}

	s.change(BackupStorageExternal, monad.None[TargetID](), access, BackupStorageReady)
}

func (s *BackupStorage) KeepLocalCopies(keep bool) {
	if s.keepLocal == keep {
		return
	}

	evt := s.changed()
	evt.KeepLocal = keep
	s.apply(evt)
}

// True once the settings were saved at least once.
func (s BackupStorage) Saved() bool { return !s.version.IsZero() }

// Records the outcome of setting up the built-in storage, if the settings did
// not change since the job started.
func (s *BackupStorage) BuiltinConfigured(version time.Time, endpoint string, useSSL bool, err error) {
	if s.mode != BackupStorageBuiltin || !s.version.Equal(version) {
		return
	}

	evt := s.changed()
	evt.Version = s.version // Same settings, only the outcome

	if err != nil {
		evt.Status = BackupStorageFailed
		evt.ErrorCode.Set(errorCode(err))
	} else {
		evt.Status = BackupStorageReady
		evt.ErrorCode.Unset()
		evt.Access.Endpoint = endpoint
		evt.Access.UseSSL = useSSL
	}

	s.apply(evt)
}

// The access to use for copies, none when they stay in the data folder.
func (s BackupStorage) Remote() (monad.Maybe[BackupStorageAccess], error) {
	switch {
	case s.mode == BackupStorageLocal:
		return monad.None[BackupStorageAccess](), nil
	case s.status != BackupStorageReady:
		return monad.None[BackupStorageAccess](), ErrStorageNotReady
	default:
		return monad.Value(s.access), nil
	}
}

func (s BackupStorage) Mode() BackupStorageMode        { return s.mode }
func (s BackupStorage) Target() monad.Maybe[TargetID]  { return s.target }
func (s BackupStorage) Access() BackupStorageAccess    { return s.access }
func (s BackupStorage) KeepLocal() bool                { return s.keepLocal }
func (s BackupStorage) Status() BackupStorageStatus    { return s.status }
func (s BackupStorage) ErrorCode() monad.Maybe[string] { return s.errorCode }
func (s BackupStorage) Version() time.Time             { return s.version }

func (s *BackupStorage) change(mode BackupStorageMode, target monad.Maybe[TargetID], access BackupStorageAccess, status BackupStorageStatus) {
	evt := s.changed()
	evt.Previous = s.target

	// Moving the built-in storage: its archives are copied from the old target first
	if s.mode == BackupStorageBuiltin && s.status == BackupStorageReady && mode == BackupStorageBuiltin && s.target != target {
		evt.PreviousAccess.Set(s.access)
	}

	evt.Mode = mode
	evt.Target = target
	evt.Access = access
	evt.Status = status
	evt.ErrorCode.Unset()
	s.apply(evt)
}

func (s *BackupStorage) changed() BackupStorageChanged {
	return BackupStorageChanged{
		Mode:      s.mode,
		Target:    s.target,
		Access:    s.access,
		KeepLocal: s.keepLocal,
		Status:    s.status,
		ErrorCode: s.errorCode,
		Version:   time.Now().UTC(),
		Previous:  s.target,
	}
}

func (s *BackupStorage) apply(e event.Event) {
	if v, ok := e.(BackupStorageChanged); ok {
		s.mode = v.Mode
		s.target = v.Target
		s.access = v.Access
		s.keepLocal = v.KeepLocal
		s.status = v.Status
		s.errorCode = v.ErrorCode
		s.version = v.Version
	}

	event.Store(s, e)
}

// Endpoints are host:port; a pasted URL loses its scheme and path.
func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint, _, _ = strings.Cut(endpoint, "/")
	return endpoint
}
