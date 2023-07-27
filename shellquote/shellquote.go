/*
Package shellquote quotes strings for shell scripts.

Sometimes you get strings from the internet and need to quote them for security,
other times you'll need to quote your own strings because doing it by hand is
just too much work.

Another option is http://github.com/kballard/go-shellquote.  The quoting algorithms are
completely different and the results vary as well, but both produce working
results in my brief testing.  See
https://github.com/frioux/go-scraps/tree/master/cmd/quotetest for a tool that
shows the results of quoting with each package.
*/
package shellquote

import (
	"errors"
	"regexp"
	"strings"
)

// QuoteMust will return a shell quoted string for the passed tokens.
func QuoteMust(in ...string) string {
	result, err := Quote(in...)
	if err != nil {
		panic(err)
	}

	return result
}

// ErrNull will be returned from Quote if any of the strings contains a null byte.
var ErrNull = errors.New("no way to quote string containing null bytes")

// Quote will return a shell quoted string for the passed tokens.
func Quote(in ...string) (string, error) {
	tmp := make([]string, len(in))
	var sawNonEqual bool
	for i, x := range in {
		if x == "" {
			tmp[i] = `''`
			continue
		}
		if strings.Contains(x, "\x00") {
			return "", ErrNull
		}

		var escape bool
		hasEqual := strings.Contains(x, "=")
		if hasEqual {
			if !sawNonEqual {
				escape = true
			}
		} else {
			sawNonEqual = true
		}

		toEsc := regexp.MustCompile(`[^\w!%+,\-./:=@^]`)
		if !escape && toEsc.MatchString(x) {
			escape = true
		}

		if escape || (!sawNonEqual && hasEqual) {
			y := strings.ReplaceAll(x, `'`, `'\''`)

			simplifyRe := regexp.MustCompile(`(?:'\\''){2,}`)
			y = simplifyRe.ReplaceAllStringFunc(y, func(str string) string {
				var inner string
				for i := 0; i < len(str)/4; i++ {
					inner += "'"
				}
				return `'"` + inner + `"'`
			})

			y = `'` + y + `'`
			y = strings.TrimSuffix(y, `''`)
			y = strings.TrimPrefix(y, `''`)

			tmp[i] = y
			continue
		}
		tmp[i] = x
	}

	return strings.Join(tmp, " "), nil
}
