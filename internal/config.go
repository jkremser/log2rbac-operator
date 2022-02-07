package internal

// Config is top-lvl entrypoint to the configuration of log2rbac operator
type Config struct {
	Log        *LogConfig
	Controller *ControllerConfig
}

// LogConfig contains log related configuration
type LogConfig struct {
	Colors   bool `env:"COLORS,default=true"`
	NoBanner bool `env:"NO_BANNER,default=false"`
}

// ControllerConfig contains internal parameters for the controller like sync intervals
type ControllerConfig struct {
	// sync intervals
	SyncIntervalAfterNoRbacEntryMinutes int `env:"SYNC_INTERVAL_AFTER_NO_RBAC_ENTRY_MINUTES,default=5"`
	SyncIntervalAfterNoLogsSeconds      int `env:"SYNC_INTERVAL_AFTER_NO_LOGS_SECONDS,default=30"`
	SyncIntervalAfterPodRestartSeconds  int `env:"SYNC_INTERVAL_AFTER_POD_RESTART_SECONDS,default=20"`
	SyncIntervalAfterErrorMinutes       int `env:"SYNC_INTERVAL_AFTER_ERROR,default=3"`

	ShouldRestartAppPods bool `env:"SHOULD_RESTART_APP_PODS,default=true"`
}
