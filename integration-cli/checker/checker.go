// Package checker provides Docker specific implementations of the go-check.Checker interface.
package checker // import "github.com/docker/docker/integration-cli/checker"

import (
	"fmt"

	"github.com/vdemeester/shakers"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

// As a commodity, we bring all check.Checker variables into the current namespace to avoid having
// to think about check.X versus checker.X.
var (
	GreaterThan = shakers.GreaterThan
)

type Compare func(x interface{}) assert.BoolOrComparison

func False(x interface{}) assert.BoolOrComparison {
	return !x.(bool)
}

func True(x interface{}) assert.BoolOrComparison {
	return x
}

func Equals(y interface{}) Compare {
	return func(x interface{}) assert.BoolOrComparison {
		return cmp.Equal(x, y)
	}
}

func Contains(y interface{}) Compare {
	return func(x interface{}) assert.BoolOrComparison {
		return cmp.Contains(x, y)
	}
}

func Not(c Compare) Compare {
	return func(x interface{}) assert.BoolOrComparison {
		r := c(x)
		switch r := r.(type) {
		case bool:
			return !r
		case cmp.Comparison:
			return !r().Success()
		default:
			panic(fmt.Sprintf("unexpected type %T", r))
		}
	}
}

func DeepEquals(y interface{}) Compare {
	return func(x interface{}) assert.BoolOrComparison {
		return cmp.DeepEqual(x, y)
	}
}

func HasLen(y int) Compare {
	return func(x interface{}) assert.BoolOrComparison {
		return cmp.Len(x, y)
	}
}

func IsNil(x interface{}) assert.BoolOrComparison {
	return cmp.Nil(x)
}

var NotNil Compare = Not(IsNil)
