package shell

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/stretchr/testify/assert"
)

func TestReduce(t *testing.T) {
	verifyCycleError := func(err error) bool {
		// Note that due to property of map, we can't predict which element in the cycle will be picked out first
		return strings.HasPrefix(err.Error(), "Cycle detected for key:")
	}
	testCases := []struct {
		original map[string]*domain.AbstractString
		expected map[string]*domain.AbstractString
		errFunc  func(error) bool
	}{
		// 0: empty
		{
			original: map[string]*domain.AbstractString{},
			expected: map[string]*domain.AbstractString{},
		},

		// 1: 1 var
		{
			original: map[string]*domain.AbstractString{
				"a": domain.Abstract("value1"),
			},
			expected: map[string]*domain.AbstractString{
				"a": domain.Abstract("value1"),
			},
		},

		// 2: 3 vars, with unresolvable symbol
		{
			original: map[string]*domain.AbstractString{
				"a": domain.Abstract("value1"),
				"b": domain.Abstract("value2"),
				"c": domain.Abstract("${a} ${b} ${a} ${b}${b} ${dne}"),
			},
			expected: map[string]*domain.AbstractString{
				"a": domain.Abstract("value1"),
				"b": domain.Abstract("value2"),
				"c": domain.Abstract("value1 value2 value1 value2value2 ${dne}"),
			},
		},

		// 3: nested
		{
			original: map[string]*domain.AbstractString{
				"zero": domain.Abstract("0"),
				"one":  domain.Abstract("1"),
				"a":    domain.Abstract("a"),
				"a_0":  domain.Abstract("${${a}${zero}}"),
				"a_1":  domain.Abstract("${${a}${one}}"),
				"a0":   domain.Abstract("ValueA_0"),
				"a1":   domain.Abstract("ValueA_1"),
			},
			expected: map[string]*domain.AbstractString{
				"zero": domain.Abstract("0"),
				"one":  domain.Abstract("1"),
				"a":    domain.Abstract("a"),
				"a_0":  domain.Abstract("ValueA_0"),
				"a_1":  domain.Abstract("ValueA_1"),
				"a0":   domain.Abstract("ValueA_0"),
				"a1":   domain.Abstract("ValueA_1"),
			},
		},

		// 4: Self reference
		{
			original: map[string]*domain.AbstractString{
				"c": domain.Abstract("${c}"),
			},
			expected: map[string]*domain.AbstractString{},
			errFunc:  verifyCycleError,
		},

		// 4: 2-cycle
		{
			original: map[string]*domain.AbstractString{
				"a": domain.Abstract("${b}"),
				"b": domain.Abstract("${a}"),
				"c": domain.Abstract("c"),
			},
			expected: map[string]*domain.AbstractString{},
			errFunc:  verifyCycleError,
		},

		// 5: 5-cycle
		{
			original: map[string]*domain.AbstractString{
				"a": domain.Abstract("${d}"),
				"b": domain.Abstract("${e}"),
				"c": domain.Abstract("${a}"),
				"d": domain.Abstract("${b}"),
				"e": domain.Abstract("${c}"),
			},
			expected: map[string]*domain.AbstractString{},
			errFunc:  verifyCycleError,
		},

		// 6: complex nesting
		{
			original: map[string]*domain.AbstractString{
				"a": domain.Abstract("${${${${${${${${${b}}}}}}}}}"),
				"b": domain.Abstract("c"),
				"c": domain.Abstract("b"),
			},
			expected: map[string]*domain.AbstractString{},
			errFunc: func(err error) bool {
				return err.Error() == "Variable nested for too many levels"
			},
		},
	}
	for _, tc := range testCases {
		err := reduceEnv(tc.original)
		if tc.errFunc == nil {
			assert.Nil(t, err, "Unexpected error: %s", err)
			assert.Equal(t, len(tc.expected), len(tc.original))
			for k, v := range tc.expected {
				altered := tc.original[k]
				assert.Equal(t, v, altered)
			}
		} else {
			if assert.NotNil(t, err, "There should be an error") {
				assert.True(t, tc.errFunc(err), fmt.Sprintf("Unexpected error: %s", err))
			} else {
				fmt.Print("Reduced map:\n")
				for k, v := range tc.original {
					fmt.Fprintf(os.Stderr, "%s: %v\n", k, v.DisplayValue())
				}
			}
		}
	}
}
