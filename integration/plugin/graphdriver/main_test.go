package graphdriver // import "github.com/tiborvass/docker/integration/plugin/graphdriver"

import (
	"fmt"
	"os"
	"testing"

	"github.com/tiborvass/docker/pkg/reexec"
	"github.com/tiborvass/docker/testutil/environment"
)

var (
	testEnv *environment.Execution
)

func init() {
	reexec.Init() // This is required for external graphdriver tests
}

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
