package graph

// Secret defines a wrapper to translate Azure Key Vault secrets to environment variables.
type Secret struct {
	Akv       string `toml:"akv,omitempty"`
	SecretEnv string `toml:"secretEnv,omitempty"`
}
