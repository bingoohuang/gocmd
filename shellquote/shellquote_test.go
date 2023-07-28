package shellquote_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bingoohuang/cmd/shellquote"
	"github.com/google/go-cmp/cmp"
)

func test(t *testing.T, in []string, expected string) {
	ret, err := shellquote.Quote(in...)
	if err != nil {
		t.Errorf("Quote errored: %s", err)
		return
	}
	Equal(t, ret, expected, "wrong quote")
}

func TestShellQuote(t *testing.T) {
	t.Parallel()

	test(t, []string{""}, `''`)
	test(t, []string{"foo"}, `foo`)
	test(t, []string{"foo", "bar"}, `foo bar`)
	test(t, []string{"foo*"}, `'foo*'`)
	test(t, []string{"foo bar"}, `'foo bar'`)
	test(t, []string{"foo'bar"}, `'foo'\''bar'`)
	test(t, []string{"'foo"}, `\''foo'`)
	test(t, []string{"foo", "bar*"}, `foo 'bar*'`)
	test(t, []string{"foo'foo", "bar", "baz'"}, `'foo'\''foo' bar 'baz'\'`)
	test(t, []string{`\`}, `'\'`)
	test(t, []string{"'"}, `\'`)
	test(t, []string{`\'`}, `'\'\'`)
	test(t, []string{"a''b"}, `'a'"''"'b'`)
	test(t, []string{"azAZ09_!%+,-./:@^"}, `azAZ09_!%+,-./:@^`)
	test(t, []string{"foo=bar", "command"}, `'foo=bar' command`)
	test(t, []string{"foo=bar", "baz=quux", "command"}, `'foo=bar' 'baz=quux' command`)

	_, err := shellquote.Quote("\x00")
	if err != shellquote.ErrNull {
		t.Errorf("err should be ErrNull; was %s", err)
	}
}

// Equal takes t, got, expected, and a prefix, returning true if got and
// expected are expected.
func Equal(t *testing.T, got, expected interface{}, prefix string, opts ...cmp.Option) bool {
	t.Helper()
	if diff := cmp.Diff(expected, got, opts...); diff != "" {
		t.Errorf("%s (-want +got):\n%s", prefix, diff)
		return false
	}

	return true
}

// JSONEqual takes a got and expected string of json and compares the parsed values with Equal.
func JSONEqual(t *testing.T, got, expected string, prefix string, opts ...cmp.Option) bool {
	t.Helper()
	var gotValue, expectedValue interface{}
	if err := json.NewDecoder(strings.NewReader(got)).Decode(&gotValue); err != nil {
		t.Errorf("Couldn't decode got: %s", err)
		return false
	}

	if err := json.NewDecoder(strings.NewReader(expected)).Decode(&expectedValue); err != nil {
		t.Errorf("Couldn't decode expected: %s", err)
		return false
	}

	return Equal(t, gotValue, expectedValue, prefix, opts...)
}
