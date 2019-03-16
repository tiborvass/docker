package httputils // import "github.com/tiborvass/docker/api/server/httputils"
import "github.com/tiborvass/docker/errdefs"

// GetHTTPErrorStatusCode retrieves status code from error message.
//
// Deprecated: use errdefs.GetHTTPErrorStatusCode
func GetHTTPErrorStatusCode(err error) int {
	return errdefs.GetHTTPErrorStatusCode(err)
}
