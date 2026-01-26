package config

// SetTestConfig sets a test configuration. This should only be used in tests.
// Call this before any code that depends on config.Get().
func SetTestConfig(cfg *Config) {
	instance = cfg
}

// SetTestDefaults sets a minimal test configuration with default values.
// This is useful when tests need config but don't care about specific values.
func SetTestDefaults() {
	instance = &Config{
		Server: ServerConfig{
			Port: "8080",
		},
		Database: DatabaseConfig{
			URL: "postgres://test:test@localhost:5432/test",
		},
		Auth: AuthConfig{
			SecretKey: "test-secret-key-for-unit-tests-only",
		},
		AWS: AWSConfig{
			S3Bucket:    "test-bucket",
			S3URLPrefix: "https://test-bucket.s3.amazonaws.com",
		},
		CORS: CORSConfig{
			AllowedOrigins: "*",
		},
	}
}

// ResetConfig resets the configuration instance to nil.
// Use this in test cleanup to ensure tests don't affect each other.
func ResetConfig() {
	instance = nil
}
