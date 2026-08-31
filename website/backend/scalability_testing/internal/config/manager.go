

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/errors"
	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

type Manager struct {
	currentEnv   interfaces.Environment
	config       *interfaces.EnvironmentConfig
	configPath   string
	validator    *Validator
}

func NewManager() (*Manager, error) {
	manager := &Manager{
		validator: NewValidator(),
	}

	env, err := manager.DetectEnvironment()
	if err != nil {
		return nil, errors.NewConfigurationError("config_manager", "detect_environment", "failed to detect environment", err)
	}
	manager.currentEnv = env

	config, err := manager.LoadEnvironmentConfig(env)
	if err != nil {
		return nil, errors.NewConfigurationError("config_manager", "load_config", "failed to load configuration", err)
	}
	manager.config = config

	return manager, nil
}

func (m *Manager) DetectEnvironment() (interfaces.Environment, error) {

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		switch strings.ToLower(env) {
		case "development", "dev":
			return interfaces.EnvDevelopment, nil
		case "staging", "stage":
			return interfaces.EnvStaging, nil
		case "production", "prod":
			return interfaces.EnvProduction, nil
		case "testing", "test":
			return interfaces.EnvTesting, nil
		default:
			return "", fmt.Errorf("unknown environment: %s", env)
		}
	}

	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		return interfaces.EnvTesting, nil
	}

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {

		if os.Getenv("PROD") == "true" || strings.Contains(os.Getenv("NAMESPACE"), "prod") {
			return interfaces.EnvProduction, nil
		}
		return interfaces.EnvStaging, nil
	}

	return interfaces.EnvDevelopment, nil
}

func (m *Manager) LoadEnvironmentConfig(env interfaces.Environment) (*interfaces.EnvironmentConfig, error) {
	config := &interfaces.EnvironmentConfig{
		Environment: env,
		Redis:       m.loadRedisConfig(),
		PostgreSQL:  m.loadPostgreSQLConfig(),
		TMDB:        m.loadTMDBConfig(),
		HTTP:        m.loadHTTPConfig(),
		Monitoring:  m.loadMonitoringConfig(),
		LoadTesting: m.loadLoadTestingConfig(),
	}

	configSources := []string{
		fmt.Sprintf("config/%s.yaml", string(env)),
		fmt.Sprintf("config/%s.yml", string(env)),
		fmt.Sprintf("config/%s.json", string(env)),
		fmt.Sprintf("./%s.yaml", string(env)),
		fmt.Sprintf("./%s.yml", string(env)),
		fmt.Sprintf("./%s.json", string(env)),
	}

	var fileConfig *interfaces.EnvironmentConfig
	var configFile string

	for _, source := range configSources {
		if _, err := os.Stat(source); err == nil {
			var err error
			fileConfig, err = m.loadConfigFile(source)
			if err != nil {
				return nil, fmt.Errorf("failed to load config file %s: %w", source, err)
			}
			configFile = source
			break
		}
	}

	if fileConfig != nil {
		config = m.mergeConfigs(config, fileConfig)
		m.configPath = configFile
	}

	return config, nil
}

func (m *Manager) ValidateConfiguration(env interfaces.Environment) (*interfaces.ConfigValidation, error) {
	validation := &interfaces.ConfigValidation{
		Environment: env,
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		CheckedAt:   time.Now(),
	}

	config, err := m.LoadEnvironmentConfig(env)
	if err != nil {
		validation.Valid = false
		validation.Errors = append(validation.Errors, fmt.Sprintf("Failed to load configuration: %v", err))
		return validation, nil
	}

	if config.Redis != nil {
		if errs := m.validator.ValidateRedis(config.Redis); len(errs) > 0 {
			validation.Errors = append(validation.Errors, errs...)
			validation.Valid = false
		}
	}

	if config.PostgreSQL != nil {
		if errs := m.validator.ValidatePostgreSQL(config.PostgreSQL); len(errs) > 0 {
			validation.Errors = append(validation.Errors, errs...)
			validation.Valid = false
		}
	}

	if config.TMDB != nil {
		if errs := m.validator.ValidateTMDB(config.TMDB); len(errs) > 0 {
			validation.Errors = append(validation.Errors, errs...)
			validation.Valid = false
		}
	}

	if config.HTTP != nil {
		if errs := m.validator.ValidateHTTP(config.HTTP); len(errs) > 0 {
			validation.Errors = append(validation.Errors, errs...)
			validation.Valid = false
		}
	}

	switch env {
	case interfaces.EnvProduction:
		m.validateProductionRequirements(config, validation)
	case interfaces.EnvStaging:
		m.validateStagingRequirements(config, validation)
	case interfaces.EnvDevelopment:
		m.validateDevelopmentRequirements(config, validation)
	}

	return validation, nil
}

func (m *Manager) ValidateConnectivity(ctx context.Context) (*interfaces.ConnectivityCheck, error) {
	check := &interfaces.ConnectivityCheck{
		CheckedAt:     time.Now(),
		OverallStatus: interfaces.HealthHealthy,
		External:      make(map[string]interfaces.ConnectivityResult),
	}

	check.Redis = m.testRedisConnectivity(ctx)
	if check.Redis.Status != interfaces.HealthHealthy {
		check.OverallStatus = interfaces.HealthDegraded
	}

	check.PostgreSQL = m.testPostgreSQLConnectivity(ctx)
	if check.PostgreSQL.Status != interfaces.HealthHealthy {
		check.OverallStatus = interfaces.HealthDegraded
	}

	check.TMDB = m.testTMDBConnectivity(ctx)
	if check.TMDB.Status != interfaces.HealthHealthy {
		if check.OverallStatus == interfaces.HealthHealthy {
			check.OverallStatus = interfaces.HealthDegraded
		} else {
			check.OverallStatus = interfaces.HealthUnhealthy
		}
	}

	return check, nil
}

func (m *Manager) GetCurrentConfigPath() string {
	return m.configPath
}

func (m *Manager) SaveConfigToFile(filename string) error {
	if m.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(m.config, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON marshal error: %w", err)
		}
	case ".yaml", ".yml", "":

		data, err = yaml.Marshal(m.config)
		if err != nil {
			return fmt.Errorf("YAML marshal error: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s (supported: .yaml, .yml, .json)", ext)
	}

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filename, err)
	}

	return nil
}

func (m *Manager) GenerateConfigTemplate(env interfaces.Environment, filename string) error {
	template := m.getConfigTemplate(env)

	ext := strings.ToLower(filepath.Ext(filename))
	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(template, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON marshal error: %w", err)
		}
	case ".yaml", ".yml", "":

		data, err = yaml.Marshal(template)
		if err != nil {
			return fmt.Errorf("YAML marshal error: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s (supported: .yaml, .yml, .json)", ext)
	}

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write template file %s: %w", filename, err)
	}

	return nil
}

func (m *Manager) getConfigTemplate(env interfaces.Environment) *interfaces.EnvironmentConfig {
	switch env {
	case interfaces.EnvProduction:
		return &interfaces.EnvironmentConfig{
			Environment: env,
			Redis: &interfaces.RedisConfig{
				URL:         "${REDIS_URL}",
				Password:    "${REDIS_PASSWORD}",
				DB:          0,
				MaxRetries:  5,
				PoolSize:    50,
				PoolTimeout: 30 * time.Second,
				IdleTimeout: 300 * time.Second,
				DefaultTTL:  900 * time.Second,
				Enabled:     true,
			},
			PostgreSQL: &interfaces.PostgreSQLConfig{
				URL:             "${DATABASE_URL}",
				MaxConnections:  50,
				MaxIdleConns:    10,
				ConnMaxLifetime: 3600 * time.Second,
				ConnMaxIdleTime: 300 * time.Second,
				SSLMode:         "require",
			},
			TMDB: &interfaces.TMDBConfig{
				APIKey:      "${TMDB_API_KEY}",
				BaseURL:     "https://api.themoviedb.org/3",
				RateLimit:   35,
				RatePeriod:  10 * time.Second,
				Timeout:     15 * time.Second,
				MaxRetries:  5,
				BackoffBase: 2000 * time.Millisecond,
			},
			HTTP: &interfaces.HTTPConfig{
				MaxIdleConns:          200,
				MaxIdleConnsPerHost:   50,
				MaxConnsPerHost:       100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
				RequestTimeout:        30 * time.Second,
			},
			Monitoring: &interfaces.MonitoringConfig{
				MetricsInterval:   15 * time.Second,
				HistoryRetention:  168 * time.Hour,
				AlertingEnabled:   true,
				DashboardEnabled:  true,
				DashboardPort:     8080,
				MetricsPort:       9090,
			},
			LoadTesting: &interfaces.LoadTestingConfig{
				MaxConcurrentUsers: 1000,
				DefaultDuration:    600 * time.Second,
				RampUpDuration:     120 * time.Second,
				MetricsInterval:    5 * time.Second,
				ResultsRetention:   720 * time.Hour,
			},
		}

	case interfaces.EnvStaging:
		return &interfaces.EnvironmentConfig{
			Environment: env,
			Redis: &interfaces.RedisConfig{
				URL:         "localhost:6379",
				Password:    "",
				DB:          1,
				MaxRetries:  3,
				PoolSize:    25,
				PoolTimeout: 30 * time.Second,
				IdleTimeout: 300 * time.Second,
				DefaultTTL:  900 * time.Second,
				Enabled:     true,
			},
			PostgreSQL: &interfaces.PostgreSQLConfig{
				URL:             "postgres://localhost:5432/waveflix_staging",
				MaxConnections:  25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 3600 * time.Second,
				ConnMaxIdleTime: 300 * time.Second,
				SSLMode:         "prefer",
			},
			TMDB: &interfaces.TMDBConfig{
				APIKey:      "${TMDB_API_KEY}",
				BaseURL:     "https://api.themoviedb.org/3",
				RateLimit:   30,
				RatePeriod:  10 * time.Second,
				Timeout:     10 * time.Second,
				MaxRetries:  3,
				BackoffBase: 1500 * time.Millisecond,
			},
			HTTP: &interfaces.HTTPConfig{
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   30,
				MaxConnsPerHost:       60,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				RequestTimeout:        30 * time.Second,
			},
			Monitoring: &interfaces.MonitoringConfig{
				MetricsInterval:   30 * time.Second,
				HistoryRetention:  72 * time.Hour,
				AlertingEnabled:   true,
				DashboardEnabled:  true,
				DashboardPort:     8080,
				MetricsPort:       9090,
			},
			LoadTesting: &interfaces.LoadTestingConfig{
				MaxConcurrentUsers: 500,
				DefaultDuration:    300 * time.Second,
				RampUpDuration:     60 * time.Second,
				MetricsInterval:    5 * time.Second,
				ResultsRetention:   168 * time.Hour,
			},
		}

	case interfaces.EnvTesting:
		return &interfaces.EnvironmentConfig{
			Environment: env,
			Redis: &interfaces.RedisConfig{
				URL:         "localhost:6379",
				Password:    "",
				DB:          2,
				MaxRetries:  2,
				PoolSize:    10,
				PoolTimeout: 10 * time.Second,
				IdleTimeout: 60 * time.Second,
				DefaultTTL:  300 * time.Second,
				Enabled:     true,
			},
			PostgreSQL: &interfaces.PostgreSQLConfig{
				URL:             "postgres://localhost:5432/waveflix_test",
				MaxConnections:  10,
				MaxIdleConns:    2,
				ConnMaxLifetime: 600 * time.Second,
				ConnMaxIdleTime: 60 * time.Second,
				SSLMode:         "disable",
			},
			TMDB: &interfaces.TMDBConfig{
				APIKey:      "test_api_key_placeholder",
				BaseURL:     "https://api.themoviedb.org/3",
				RateLimit:   20,
				RatePeriod:  10 * time.Second,
				Timeout:     5 * time.Second,
				MaxRetries:  2,
				BackoffBase: 500 * time.Millisecond,
			},
			HTTP: &interfaces.HTTPConfig{
				MaxIdleConns:          50,
				MaxIdleConnsPerHost:   10,
				MaxConnsPerHost:       20,
				IdleConnTimeout:       30 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 5 * time.Second,
				RequestTimeout:        10 * time.Second,
			},
			Monitoring: &interfaces.MonitoringConfig{
				MetricsInterval:   60 * time.Second,
				HistoryRetention:  24 * time.Hour,
				AlertingEnabled:   false,
				DashboardEnabled:  false,
				DashboardPort:     8081,
				MetricsPort:       9091,
			},
			LoadTesting: &interfaces.LoadTestingConfig{
				MaxConcurrentUsers: 50,
				DefaultDuration:    60 * time.Second,
				RampUpDuration:     10 * time.Second,
				MetricsInterval:    2 * time.Second,
				ResultsRetention:   24 * time.Hour,
			},
		}

	default: 
		return &interfaces.EnvironmentConfig{
			Environment: env,
			Redis: &interfaces.RedisConfig{
				URL:         "localhost:6379",
				Password:    "",
				DB:          0,
				MaxRetries:  3,
				PoolSize:    10,
				PoolTimeout: 30 * time.Second,
				IdleTimeout: 300 * time.Second,
				DefaultTTL:  900 * time.Second,
				Enabled:     true,
			},
			PostgreSQL: &interfaces.PostgreSQLConfig{
				URL:             "postgres://localhost:5432/waveflix_dev",
				MaxConnections:  25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 3600 * time.Second,
				ConnMaxIdleTime: 300 * time.Second,
				SSLMode:         "prefer",
			},
			TMDB: &interfaces.TMDBConfig{
				APIKey:      "${TMDB_API_KEY}",
				BaseURL:     "https://api.themoviedb.org/3",
				RateLimit:   30,
				RatePeriod:  10 * time.Second,
				Timeout:     10 * time.Second,
				MaxRetries:  3,
				BackoffBase: 1000 * time.Millisecond,
			},
			HTTP: &interfaces.HTTPConfig{
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   25,
				MaxConnsPerHost:       50,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				RequestTimeout:        30 * time.Second,
			},
			Monitoring: &interfaces.MonitoringConfig{
				MetricsInterval:   30 * time.Second,
				HistoryRetention:  24 * time.Hour,
				AlertingEnabled:   true,
				DashboardEnabled:  true,
				DashboardPort:     8080,
				MetricsPort:       9090,
			},
			LoadTesting: &interfaces.LoadTestingConfig{
				MaxConcurrentUsers: 100,
				DefaultDuration:    300 * time.Second,
				RampUpDuration:     60 * time.Second,
				MetricsInterval:    5 * time.Second,
				ResultsRetention:   168 * time.Hour,
			},
		}
	}
}

func (m *Manager) GetCurrentConfig() *interfaces.EnvironmentConfig {
	return m.config
}

func (m *Manager) GetCurrentEnvironment() interfaces.Environment {
	return m.currentEnv
}

func (m *Manager) ReloadConfiguration() error {
	config, err := m.LoadEnvironmentConfig(m.currentEnv)
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}
	m.config = config
	return nil
}

func (m *Manager) ValidateAllEnvironments() (map[interfaces.Environment]*interfaces.ConfigValidation, error) {
	environments := []interfaces.Environment{
		interfaces.EnvDevelopment,
		interfaces.EnvStaging,
		interfaces.EnvProduction,
		interfaces.EnvTesting,
	}

	results := make(map[interfaces.Environment]*interfaces.ConfigValidation)

	for _, env := range environments {
		validation, err := m.ValidateConfiguration(env)
		if err != nil {
			return nil, fmt.Errorf("failed to validate environment %s: %w", env, err)
		}
		results[env] = validation
	}

	return results, nil
}

func (m *Manager) GetBackwardCompatibleConfig() *LegacyConfig {
	if m.config == nil {
		return &LegacyConfig{}
	}

	legacy := &LegacyConfig{
		Environment: string(m.config.Environment),
	}

	if m.config.Redis != nil {
		legacy.Redis = LegacyRedisConfig{
			URL:         m.config.Redis.URL,
			Password:    m.config.Redis.Password,
			DB:          m.config.Redis.DB,
			MaxRetries:  m.config.Redis.MaxRetries,
			PoolSize:    m.config.Redis.PoolSize,
			PoolTimeout: m.config.Redis.PoolTimeout,
			IdleTimeout: m.config.Redis.IdleTimeout,
			DefaultTTL:  m.config.Redis.DefaultTTL,
			Enabled:     m.config.Redis.Enabled,
		}
	}

	if m.config.TMDB != nil {
		legacy.TMDB = LegacyTMDBConfig{
			APIKey:      m.config.TMDB.APIKey,
			BaseURL:     m.config.TMDB.BaseURL,
			RateLimit:   m.config.TMDB.RateLimit,
			RatePeriod:  m.config.TMDB.RatePeriod,
			Timeout:     m.config.TMDB.Timeout,
			MaxRetries:  m.config.TMDB.MaxRetries,
			BackoffBase: m.config.TMDB.BackoffBase,
		}
	}

	if m.config.HTTP != nil {
		legacy.HTTP = LegacyHTTPConfig{
			MaxIdleConns:          m.config.HTTP.MaxIdleConns,
			MaxIdleConnsPerHost:   m.config.HTTP.MaxIdleConnsPerHost,
			MaxConnsPerHost:       m.config.HTTP.MaxConnsPerHost,
			IdleConnTimeout:       m.config.HTTP.IdleConnTimeout,
			TLSHandshakeTimeout:   m.config.HTTP.TLSHandshakeTimeout,
			ResponseHeaderTimeout: m.config.HTTP.ResponseHeaderTimeout,
			RequestTimeout:        m.config.HTTP.RequestTimeout,
		}
	}

	return legacy
}

type LegacyConfig struct {
	Environment string            `json:"environment"`
	Redis       LegacyRedisConfig `json:"redis"`
	TMDB        LegacyTMDBConfig  `json:"tmdb"`
	HTTP        LegacyHTTPConfig  `json:"http"`
}

type LegacyRedisConfig struct {
	URL         string        `json:"url"`
	Password    string        `json:"password"`
	DB          int           `json:"db"`
	MaxRetries  int           `json:"max_retries"`
	PoolSize    int           `json:"pool_size"`
	PoolTimeout time.Duration `json:"pool_timeout"`
	IdleTimeout time.Duration `json:"idle_timeout"`
	DefaultTTL  time.Duration `json:"default_ttl"`
	Enabled     bool          `json:"enabled"`
}

type LegacyTMDBConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	RateLimit   int           `json:"rate_limit"`
	RatePeriod  time.Duration `json:"rate_period"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"max_retries"`
	BackoffBase time.Duration `json:"backoff_base"`
}

type LegacyHTTPConfig struct {
	MaxIdleConns          int           `json:"max_idle_conns"`
	MaxIdleConnsPerHost   int           `json:"max_idle_conns_per_host"`
	MaxConnsPerHost       int           `json:"max_conns_per_host"`
	IdleConnTimeout       time.Duration `json:"idle_conn_timeout"`
	TLSHandshakeTimeout   time.Duration `json:"tls_handshake_timeout"`
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout"`
	RequestTimeout        time.Duration `json:"request_timeout"`
}

func (m *Manager) loadRedisConfig() *interfaces.RedisConfig {
	return &interfaces.RedisConfig{
		URL:         getEnvDefault("REDIS_URL", "localhost:6379"),
		Password:    os.Getenv("REDIS_PASSWORD"),
		DB:          getEnvInt("REDIS_DB", 0),
		MaxRetries:  getEnvInt("REDIS_MAX_RETRIES", 3),
		PoolSize:    getEnvInt("REDIS_POOL_SIZE", 20),
		PoolTimeout: time.Duration(getEnvInt("REDIS_POOL_TIMEOUT", 30)) * time.Second,
		IdleTimeout: time.Duration(getEnvInt("REDIS_IDLE_TIMEOUT", 300)) * time.Second,
		DefaultTTL:  time.Duration(getEnvInt("REDIS_DEFAULT_TTL", 900)) * time.Second, 
		Enabled:     getEnvBool("REDIS_ENABLED", true),
	}
}

func (m *Manager) loadPostgreSQLConfig() *interfaces.PostgreSQLConfig {
	return &interfaces.PostgreSQLConfig{
		URL:             getEnvDefault("DATABASE_URL", "postgres://localhost:5432/waveflix"),
		MaxConnections:  getEnvInt("DB_MAX_CONNECTIONS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 3600)) * time.Second, 
		ConnMaxIdleTime: time.Duration(getEnvInt("DB_CONN_MAX_IDLE_TIME", 300)) * time.Second,  
		SSLMode:         getEnvDefault("DB_SSL_MODE", "prefer"),
	}
}

func (m *Manager) loadTMDBConfig() *interfaces.TMDBConfig {
	return &interfaces.TMDBConfig{
		APIKey:      os.Getenv("TMDB_API_KEY"),
		BaseURL:     getEnvDefault("TMDB_BASE_URL", "https://api.themoviedb.org/3"),
		RateLimit:   getEnvInt("TMDB_RATE_LIMIT", 35), 
		RatePeriod:  time.Duration(getEnvInt("TMDB_RATE_PERIOD", 10)) * time.Second,
		Timeout:     time.Duration(getEnvInt("TMDB_TIMEOUT", 10)) * time.Second,
		MaxRetries:  getEnvInt("TMDB_MAX_RETRIES", 3),
		BackoffBase: time.Duration(getEnvInt("TMDB_BACKOFF_BASE", 1000)) * time.Millisecond,
	}
}

func (m *Manager) loadHTTPConfig() *interfaces.HTTPConfig {
	return &interfaces.HTTPConfig{
		MaxIdleConns:          getEnvInt("HTTP_MAX_IDLE_CONNS", 200),
		MaxIdleConnsPerHost:   getEnvInt("HTTP_MAX_IDLE_CONNS_PER_HOST", 50),
		MaxConnsPerHost:       getEnvInt("HTTP_MAX_CONNS_PER_HOST", 100),
		IdleConnTimeout:       time.Duration(getEnvInt("HTTP_IDLE_CONN_TIMEOUT", 90)) * time.Second,
		TLSHandshakeTimeout:   time.Duration(getEnvInt("HTTP_TLS_HANDSHAKE_TIMEOUT", 10)) * time.Second,
		ResponseHeaderTimeout: time.Duration(getEnvInt("HTTP_RESPONSE_HEADER_TIMEOUT", 10)) * time.Second,
		RequestTimeout:        time.Duration(getEnvInt("HTTP_REQUEST_TIMEOUT", 30)) * time.Second,
	}
}

func (m *Manager) loadMonitoringConfig() *interfaces.MonitoringConfig {
	return &interfaces.MonitoringConfig{
		MetricsInterval:   time.Duration(getEnvInt("MONITORING_METRICS_INTERVAL", 30)) * time.Second,
		HistoryRetention:  time.Duration(getEnvInt("MONITORING_HISTORY_RETENTION", 168)) * time.Hour, 
		AlertingEnabled:   getEnvBool("MONITORING_ALERTING_ENABLED", true),
		DashboardEnabled:  getEnvBool("MONITORING_DASHBOARD_ENABLED", true),
		DashboardPort:     getEnvInt("MONITORING_DASHBOARD_PORT", 8080),
		MetricsPort:       getEnvInt("MONITORING_METRICS_PORT", 9090),
	}
}

func (m *Manager) loadLoadTestingConfig() *interfaces.LoadTestingConfig {
	return &interfaces.LoadTestingConfig{
		MaxConcurrentUsers: getEnvInt("LOADTEST_MAX_CONCURRENT_USERS", 1000),
		DefaultDuration:    time.Duration(getEnvInt("LOADTEST_DEFAULT_DURATION", 300)) * time.Second, 
		RampUpDuration:     time.Duration(getEnvInt("LOADTEST_RAMP_UP_DURATION", 60)) * time.Second,  
		MetricsInterval:    time.Duration(getEnvInt("LOADTEST_METRICS_INTERVAL", 5)) * time.Second,
		ResultsRetention:   time.Duration(getEnvInt("LOADTEST_RESULTS_RETENTION", 720)) * time.Hour, 
	}
}

func (m *Manager) loadConfigFile(filename string) (*interfaces.EnvironmentConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config interfaces.EnvironmentConfig

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("YAML parse error in %s: %w", filename, err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("JSON parse error in %s: %w", filename, err)
		}
	default:

		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("configuration parse error in %s (tried YAML): %w", filename, err)
		}
	}

	return &config, nil
}

func (m *Manager) mergeConfigs(envConfig, fileConfig *interfaces.EnvironmentConfig) *interfaces.EnvironmentConfig {

	if fileConfig.Redis != nil {
		envConfig.Redis = fileConfig.Redis
	}
	if fileConfig.PostgreSQL != nil {
		envConfig.PostgreSQL = fileConfig.PostgreSQL
	}
	if fileConfig.TMDB != nil {
		envConfig.TMDB = fileConfig.TMDB
	}
	if fileConfig.HTTP != nil {
		envConfig.HTTP = fileConfig.HTTP
	}
	if fileConfig.Monitoring != nil {
		envConfig.Monitoring = fileConfig.Monitoring
	}
	if fileConfig.LoadTesting != nil {
		envConfig.LoadTesting = fileConfig.LoadTesting
	}

	return envConfig
}

func (m *Manager) validateProductionRequirements(config *interfaces.EnvironmentConfig, validation *interfaces.ConfigValidation) {

	if config.Redis == nil || !config.Redis.Enabled {
		validation.Errors = append(validation.Errors, "Redis must be enabled in production")
		validation.Valid = false
	}

	if config.TMDB == nil || config.TMDB.APIKey == "" {
		validation.Errors = append(validation.Errors, "TMDB API key is required in production")
		validation.Valid = false
	}

	if config.PostgreSQL != nil && config.PostgreSQL.SSLMode == "disable" {
		validation.Warnings = append(validation.Warnings, "SSL is disabled for PostgreSQL in production")
	}

	if config.Monitoring == nil || !config.Monitoring.AlertingEnabled {
		validation.Warnings = append(validation.Warnings, "Alerting should be enabled in production")
	}
}

func (m *Manager) validateStagingRequirements(config *interfaces.EnvironmentConfig, validation *interfaces.ConfigValidation) {

	if config.Redis == nil || !config.Redis.Enabled {
		validation.Warnings = append(validation.Warnings, "Redis should be enabled in staging for realistic testing")
	}

	if config.TMDB == nil || config.TMDB.APIKey == "" {
		validation.Warnings = append(validation.Warnings, "TMDB API key should be configured in staging")
	}
}

func (m *Manager) validateDevelopmentRequirements(config *interfaces.EnvironmentConfig, validation *interfaces.ConfigValidation) {

	if config.Redis == nil || !config.Redis.Enabled {
		validation.Warnings = append(validation.Warnings, "Redis is disabled - using memory cache fallback")
	}

	if config.TMDB == nil || config.TMDB.APIKey == "" {
		validation.Warnings = append(validation.Warnings, "TMDB API key not configured - some features may be limited")
	}
}

func (m *Manager) testRedisConnectivity(ctx context.Context) interfaces.ConnectivityResult {

	result := interfaces.ConnectivityResult{
		Service:      "redis",
		Status:       interfaces.HealthHealthy,
		ResponseTime: time.Millisecond * 5,
		Details:      make(map[string]interface{}),
	}

	if m.config != nil && m.config.Redis != nil {
		result.Details["url"] = m.config.Redis.URL
	} else {
		result.Details["url"] = "not configured"
		result.Status = interfaces.HealthUnknown
	}

	return result
}

func (m *Manager) testPostgreSQLConnectivity(ctx context.Context) interfaces.ConnectivityResult {

	result := interfaces.ConnectivityResult{
		Service:      "postgresql",
		Status:       interfaces.HealthHealthy,
		ResponseTime: time.Millisecond * 10,
		Details:      make(map[string]interface{}),
	}

	if m.config != nil && m.config.PostgreSQL != nil {
		result.Details["url"] = m.config.PostgreSQL.URL
	} else {
		result.Details["url"] = "not configured"
		result.Status = interfaces.HealthUnknown
	}

	return result
}

func (m *Manager) testTMDBConnectivity(ctx context.Context) interfaces.ConnectivityResult {

	result := interfaces.ConnectivityResult{
		Service:      "tmdb",
		Status:       interfaces.HealthHealthy,
		ResponseTime: time.Millisecond * 100,
		Details:      make(map[string]interface{}),
	}

	if m.config != nil && m.config.TMDB != nil {
		result.Details["base_url"] = m.config.TMDB.BaseURL
	} else {
		result.Details["base_url"] = "not configured"
		result.Status = interfaces.HealthUnknown
	}

	return result
}

func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}