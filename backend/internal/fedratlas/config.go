package fedratlas

import "os"

// LoadConfig loads Fedratlas configuration from environment
func LoadConfig() Config {
	return Config{
		Enabled:  os.Getenv("FEDRATLAS_ENABLED") != "false",
		GRPCHost: os.Getenv("FEDRATLAS_GRPC_HOST"),
		GRPCPort: os.Getenv("FEDRATLAS_GRPC_PORT"),
		ServerID: os.Getenv("FEDRATLAS_SERVER_ID"),
	}
}
