
package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/hamin-baek/waveflix-hub/scalability-testing/internal/interfaces"
)

func TestNewManager(t *testing.T) {

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	env := manager.GetCurrentEnvironment()
	if env == "" {
		t.Error("Environment should not be empty")
	}

	config := manager.GetCurrentConfig()
	if config == nil {
		t.Error("Configuration should not be nil")
	}
}

func TestDetectEnvironment(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		name     string
		envVars  map[string]string
		expected interfaces.Environment
	}{
		{
			name:     "explicit development",
			envVars:  map[string]string{"ENVIRONMENT": "development"},
			expected: interfaces.EnvDevelopment,
		},
		{
			name:     "explicit production",
			envVars:  map[string]string{"ENVIRONMENT": "production"},
			expected: interfaces.EnvProduction,
		},
		{
			name:     "explicit staging", 
			envVars:  map[string]string{"ENVIRONMENT": "staging"},
			expected: interfaces.EnvStaging,
		},
		{
			name:     "CI environment",
			envVars:  map[string]string{"CI": "true"},
			expected: interfaces.EnvTesting,
		},
		{
			name:     "GitHub Actions",
			envVars:  map[string]string{"GITHUB_ACTIONS": "true"},
			expected: interfaces.EnvTesting,
		},
		{
			name:     "Kubernetes production",
			envVars:  map[string]string{"KUBERNETES_SERVICE_HOST": "10.0.0.1", "PROD": "true"},
			expected: interfaces.EnvProduction,
		},
		{
			name:     "Kubernetes staging",
			envVars:  map[string]string{"KUBERNETES_SERVICE_HOST": "10.0.0.1"},
			expected: interfaces.EnvStaging,
		},
		{
			name:     "default development",
			envVars:  map[string]string{},
			expected: interfaces.EnvDevelopment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			clearEnv()

			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			env, err := manager.DetectEnvironment()
			if err != nil {
				t.Errorf("DetectEnvironment() error = %v", err)
				return
			}

			if env != tt.expected {
				t.Errorf("DetectEnvironment() = %v, want %v", env, tt.expected)
			}
		})
	}
}

func TestLoadEnvironmentConfig(t *testing.T) {
	manager := &Manager{}

	environments := []interfaces.Environment{
		interfaces.EnvDevelopment,
		interfaces.EnvStaging,
		interfaces.EnvProduction,
		interfaces.EnvTesting,
	}

	for _, env := range environments {
		t.Run(string(env), func(t *testing.T) {
			config, err := manager.LoadEnvironmentConfig(env)
			if err != nil {
				t.Errorf("LoadEnvironmentConfig(%v) error = %v", env, err)
				return
			}

			if config == nil {
				t.Error("Configuration should not be nil")
				return
			}

			if config.Environment != env {
				t.Errorf("Configuration environment = %v, want %v", config.Environment, env)
			}

			if config.Redis == nil {
				t.Error("Redis configuration should not be nil")
			}
			if config.PostgreSQL == nil {
				t.Error("PostgreSQL configuration should not be nil")
			}
			if config.TMDB == nil {
				t.Error("TMDB configuration should not be nil")
			}
			if config.HTTP == nil {
				t.Error("HTTP configuration should not be nil")
			}
			if config.Monitoring == nil {
				t.Error("Monitoring configuration should not be nil")
			}
			if config.LoadTesting == nil {
				t.Error("Load testing configuration should not be nil")
			}
		})
	}
}

func TestConfigurationFileSupport(t *testing.T) {
	manager := &Manager{}
	tempDir := t.TempDir()

	t.Run("YAML configuration", func(t *testing.T) {
		yamlConfig := &interfaces.EnvironmentConfig{
			Environment: interfaces.EnvDevelopment,
			Redis: &interfaces.RedisConfig{
				URL:     "localhost:6380",
				Enabled: true,
			},
		}

		yamlFile := filepath.Join(tempDir, "test.yaml")
		yamlData, _ := yaml.Marshal(yamlConfig)
		os.WriteFile(yamlFile, yamlData, 0644)

		config, err := manager.loadConfigFile(yamlFile)
		if err != nil {
			t.Errorf("Failed to load YAML config: %v", err)
		}

		if config.Redis.URL != "localhost:6380" {
			t.Errorf("YAML config not loaded correctly, got URL: %s", config.Redis.URL)
		}
	})

	t.Run("JSON configuration", func(t *testing.T) {
		jsonConfig := &interfaces.EnvironmentConfig{
			Environment: interfaces.EnvStaging,
			Redis: &interfaces.RedisConfig{
				URL:     "localhost:6381",
				Enabled: true,
			},
		}

		jsonFile := filepath.Join(tempDir, "test.json")
		jsonData, _ := json.Marshal(jsonConfig)
		os.WriteFile(jsonFile, jsonData, 0644)

		config, err := manager.loadConfigFile(jsonFile)
		if err != nil {
			t.Errorf("Failed to load JSON config: %v", err)
		}

		if config.Redis.URL != "localhost:6381" {
			t.Errorf("JSON config not loaded correctly, got URL: %s", config.Redis.URL)
		}
	})
}

func TestValidateConfiguration(t *testing.T) {
	manager := &Manager{validator: NewValidator()}

	environments := []interfaces.Environment{
		interfaces.EnvDevelopment,
		interfaces.EnvStaging,
		interfaces.EnvProduction,
		interfaces.EnvTesting,
	}

	for _, env := range environments {
		t.Run(string(env), func(t *testing.T) {
			validation, err := manager.ValidateConfiguration(env)
			if err != nil {
				t.Errorf("ValidateConfiguration(%v) error = %v", env, err)
				return
			}

			if validation == nil {
				t.Error("Validation result should not be nil")
				return
			}

			if validation.Environment != env {
				t.Errorf("Validation environment = %v, want %v", validation.Environment, env)
			}

			if len(validation.Errors) == 0 && len(validation.Warnings) == 0 {

				t.Logf("No errors or warnings for environment %s", env)
			}

			for _, err := range validation.Errors {
				t.Logf("Validation error for %s: %s", env, err)
			}
			for _, warn := range validation.Warnings {
				t.Logf("Validation warning for %s: %s", env, warn)
			}
		})
	}
}

func TestConnectivityCheck(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	check, err := manager.ValidateConnectivity(ctx)
	if err != nil {
		t.Logf("ConnectivityCheck error (expected in test environment): %v", err)
		return
	}

	if check == nil {
		t.Error("Connectivity check result should not be nil")
		return
	}

	t.Logf("Overall status: %s", check.OverallStatus)
	t.Logf("Redis status: %s", check.Redis.Status)
	t.Logf("PostgreSQL status: %s", check.PostgreSQL.Status)
	t.Logf("TMDB status: %s", check.TMDB.Status)
}

func TestConfigurationMerging(t *testing.T) {
	manager := &Manager{}

	envConfig := &interfaces.EnvironmentConfig{
		Environment: interfaces.EnvDevelopment,
		Redis: &interfaces.RedisConfig{
			URL:     "localhost:6379",
			Enabled: true,
		},
		TMDB: &interfaces.TMDBConfig{
			APIKey: "env_key",
		},
	}

	fileConfig := &interfaces.EnvironmentConfig{
		Redis: &interfaces.RedisConfig{
			URL:     "file.redis.com:6379",
			Enabled: false,
		},

	}

	merged := manager.mergeConfigs(envConfig, fileConfig)

	if merged.Redis.URL != "file.redis.com:6379" {
		t.Errorf("Redis URL not overridden, got: %s", merged.Redis.URL)
	}
	if merged.Redis.Enabled != false {
		t.Error("Redis enabled should be overridden to false")
	}
	if merged.TMDB.APIKey != "env_key" {
		t.Errorf("TMDB API key should remain from env, got: %s", merged.TMDB.APIKey)
	}
}

func TestConfigurationSaving(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tempDir := t.TempDir()

	t.Run("Save as YAML", func(t *testing.T) {
		yamlFile := filepath.Join(tempDir, "output.yaml")
		err := manager.SaveConfigToFile(yamlFile)
		if err != nil {
			t.Errorf("Failed to save YAML config: %v", err)
		}

		if _, err := os.Stat(yamlFile); os.IsNotExist(err) {
			t.Error("YAML config file was not created")
		}
	})

	t.Run("Save as JSON", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "output.json")
		err := manager.SaveConfigToFile(jsonFile)
		if err != nil {
			t.Errorf("Failed to save JSON config: %v", err)
		}

		if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
			t.Error("JSON config file was not created")
		}
	})
}

func TestTemplateGeneration(t *testing.T) {
	manager := &Manager{}
	tempDir := t.TempDir()

	environments := []interfaces.Environment{
		interfaces.EnvDevelopment,
		interfaces.EnvStaging,
		interfaces.EnvProduction,
		interfaces.EnvTesting,
	}

	for _, env := range environments {
		t.Run(string(env), func(t *testing.T) {
			templateFile := filepath.Join(tempDir, string(env)+".yaml")
			err := manager.GenerateConfigTemplate(env, templateFile)
			if err != nil {
				t.Errorf("Failed to generate template for %s: %v", env, err)
			}

			if _, err := os.Stat(templateFile); os.IsNotExist(err) {
				t.Errorf("Template file for %s was not created", env)
			}

			config, err := manager.loadConfigFile(templateFile)
			if err != nil {
				t.Errorf("Failed to load generated template for %s: %v", env, err)
			}

			if config.Environment != env {
				t.Errorf("Template environment mismatch for %s", env)
			}
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	legacy := manager.GetBackwardCompatibleConfig()
	if legacy == nil {
		t.Error("Legacy config should not be nil")
	}

	if legacy.Environment == "" {
		t.Error("Legacy environment should not be empty")
	}

	if legacy.Redis.URL == "" {
		t.Error("Legacy Redis URL should not be empty")
	}

	if legacy.TMDB.BaseURL == "" {
		t.Error("Legacy TMDB base URL should not be empty")
	}
}

func TestValidateAllEnvironments(t *testing.T) {
	manager := &Manager{validator: NewValidator()}

	results, err := manager.ValidateAllEnvironments()
	if err != nil {
		t.Errorf("ValidateAllEnvironments error = %v", err)
		return
	}

	expectedEnvs := []interfaces.Environment{
		interfaces.EnvDevelopment,
		interfaces.EnvStaging,
		interfaces.EnvProduction,
		interfaces.EnvTesting,
	}

	for _, env := range expectedEnvs {
		if _, exists := results[env]; !exists {
			t.Errorf("Missing validation result for environment: %s", env)
		}
	}
}

func TestReloadConfiguration(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	originalConfig := manager.GetCurrentConfig()

	err = manager.ReloadConfiguration()
	if err != nil {
		t.Errorf("ReloadConfiguration error = %v", err)
		return
	}

	newConfig := manager.GetCurrentConfig()
	if newConfig == nil {
		t.Error("Reloaded config should not be nil")
	}

	if newConfig.Environment != originalConfig.Environment {
		t.Error("Environment should remain consistent after reload")
	}
}

func clearEnv() {
	envVars := []string{
		"ENVIRONMENT", "CI", "GITHUB_ACTIONS", "KUBERNETES_SERVICE_HOST", 
		"PROD", "NAMESPACE", "REDIS_URL", "REDIS_PASSWORD", "DATABASE_URL",
		"TMDB_API_KEY",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}
}

func BenchmarkNewManager(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager, err := NewManager()
		if err != nil {
			b.Fatalf("Failed to create manager: %v", err)
		}
		_ = manager
	}
}

func BenchmarkLoadEnvironmentConfig(b *testing.B) {
	manager := &Manager{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		config, err := manager.LoadEnvironmentConfig(interfaces.EnvDevelopment)
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
		_ = config
	}
}

func BenchmarkValidateConfiguration(b *testing.B) {
	manager := &Manager{validator: NewValidator()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validation, err := manager.ValidateConfiguration(interfaces.EnvDevelopment)
		if err != nil {
			b.Fatalf("Failed to validate config: %v", err)
		}
		_ = validation
	}
}