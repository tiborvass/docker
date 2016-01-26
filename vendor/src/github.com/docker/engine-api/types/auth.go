package types

// AuthConfig contains authorization information for connecting to a Registry
type AuthConfig struct {
	Username      string        `json:"username,omitempty"`
	Password      string        `json:"password,omitempty"`
	Auth          string        `json:"auth"`
	Email         string        `json:"email"`
	ServerAddress string        `json:"serveraddress,omitempty"`
	RegistryToken RegistryToken `json:"registrytoken,omitempty"`
}

// RegistryToken contains a host-bound token to be used for connecting to a registry.
type RegistryToken struct {
	Host  string `json:"host,omitempty"`
	Token string `json:"token,omitempty"`
}
