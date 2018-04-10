package logging // import "github.com/tiborvass/docker/integration/plugin/logging"

import (
	"fmt"
	"os"
	"testing"

	"github.com/tiborvass/docker/internal/test/environment"
)

var (
	testEnv *environment.Execution
)

func TestMain(m *testing.M) {
	var err error
	testEnv, err = environment.New()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = environment.EnsureFrozenImagesLinux(testEnv)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	testEnv.Print()
	os.Exit(m.Run())
}
