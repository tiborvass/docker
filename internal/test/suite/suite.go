package suite

import (
	"flag"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

var TimeoutFlag = flag.Duration("timeout", 0, "per-test panic after duration `d` (default 0, timeout disabled)")

var typTestingT = reflect.TypeOf(new(testing.T))

// Run takes a testing suite and runs all of the tests attached to it.
func Run(t *testing.T, suite interface{}) {
	defer failOnPanic(t)

	t.Logf("test %s", t.Name())
	suiteSetupDone := false

	methodFinder := reflect.TypeOf(suite)
	for index := 0; index < methodFinder.NumMethod(); index++ {
		method := methodFinder.Method(index)
		if !methodFilter(method.Name, method.Type) {
			//t.Logf("skipping %s", method.Name)
			continue
		}
		if !suiteSetupDone {
			if setupAllSuite, ok := suite.(SetupAllSuite); ok {
				setupAllSuite.SetUpSuite(t)
			}
			defer func() {
				if tearDownAllSuite, ok := suite.(TearDownAllSuite); ok {
					tearDownAllSuite.TearDownSuite(t)
				}
			}()
			suiteSetupDone = true
		}
		t.Run(method.Name, func(t *testing.T) {
			defer failOnPanic(t)

			// TODO: parameterize
			//t.Parallel()

			if setupTestSuite, ok := suite.(SetupTestSuite); ok {
				setupTestSuite.SetUpTest(t)
			}
			defer func() {
				if tearDownTestSuite, ok := suite.(TearDownTestSuite); ok {
					tearDownTestSuite.TearDownTest(t)
				}
			}()

			done := make(chan struct{})
			var timeout <-chan time.Time
			if *TimeoutFlag > 0 {
				timeout = time.After(*TimeoutFlag)
			}
			go func() {
				defer close(done)
				method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(t)})
			}()
			select {
			case <-done:
			case <-timeout:
				if timeoutSuite, ok := suite.(TimeoutTestSuite); ok {
					timeoutSuite.Timeout()
				}
				panic(fmt.Sprintf("test timed out after %s since start of test", *TimeoutFlag))
			}
		})
	}
}

func failOnPanic(t *testing.T) {
	r := recover()
	if r != nil {
		t.Errorf("test panicked: %v\n%s", r, debug.Stack())
		t.FailNow()
	}
}

func methodFilter(name string, typ reflect.Type) bool {
	return strings.HasPrefix(name, "Test") && typ.NumIn() == 2 && typ.In(1) == typTestingT // 2 params: method receiver and *testing.T
	/*
		if ok, _ := regexp.MatchString("^Test", name); !ok {
			return false
		}
		return true
		//return regexp.MatchString(*matchMethod, name)
	*/
}
