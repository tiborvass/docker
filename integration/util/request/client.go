package request

import (
	"testing"

	"github.com/tiborvass/docker/client"
	"github.com/stretchr/testify/require"
)

// NewAPIClient returns a docker API client configured from environment variables
func NewAPIClient(t *testing.T) client.APIClient {
	clt, err := client.NewEnvClient()
	require.NoError(t, err)
	return clt
}
