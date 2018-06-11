package system // import "github.com/tiborvass/docker/integration/system"

import (
	"context"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/integration/internal/requirement"
	"github.com/tiborvass/docker/internal/test/request"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/skip"
)

// Test case for GitHub 22244
func TestLoginFailsWithBadCredentials(t *testing.T) {
	skip.If(t, !requirement.HasHubConnectivity(t))

	client := request.NewAPIClient(t)

	config := types.AuthConfig{
		Username: "no-user",
		Password: "no-password",
	}
	_, err := client.RegistryLogin(context.Background(), config)
	expected := "Error response from daemon: Get https://registry-1.docker.io/v2/: unauthorized: incorrect username or password"
	assert.Check(t, is.Error(err, expected))
}
