package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessFeedContent(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		title  string
		body   string
		max    int
		result string
	}{
		{
			name:   "when max rune length is greater than content length then return whole content",
			title:  "Some Test",
			body:   "Oh yeah, this is cool!",
			max:    37,
			result: "**Some Test**\u2028Oh yeah, this is cool!",
		},
		{
			name:   "when max rune length is exactly the content length then return whole content",
			title:  "Some Test",
			body:   "Oh yeah, this is cool!",
			max:    36,
			result: "**Some Test**\u2028Oh yeah, this is cool!",
		},
		{
			name:   "when max rune length is smaller than content length then return truncated content",
			title:  "Some Test",
			body:   "Oh yeah, this is cool!",
			max:    35,
			result: "**Some Test**\u2028Oh yeah, this is cool ...",
		},
		{
			name:   "when max rune length is smaller than content length and content is multi-byte Unicode then return truncated content with multi-byte characters intact",
			title:  "Äöüß", // all these umlauts are two bytes long encoded in UTF-8 and would be chopped in half when counting bytes instead of runes
			body:   "truncated anyways",
			max:    5,
			result: "**Äöü ...",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			actual := ProcessFeedContent(testCase.title, testCase.body, testCase.max)
			if testCase.result != actual {
				expectedRunes := []rune(testCase.result)
				actualRunes := []rune(actual)

				hint := func() string {
					for i, expectedRune := range expectedRunes {
						if i >= len(actualRunes) {
							return fmt.Sprintf("actual too short, ends at rune index %d", i)
						}
						actualRune := actualRunes[i]
						if expectedRune != actualRune {
							return fmt.Sprintf("first difference at rune index %d:\n"+
								"expected: '%s' (%U)\n"+
								"actual:   '%s' (%U)",
								i, string(expectedRune), expectedRune,
								string(actualRune), actualRune)
						}
					}
					return fmt.Sprintf("actual too long, expected ends at rune index %d", len(expectedRunes))
				}()

				t.Logf("markdown not equal\n"+
					"expected: '%s' (byte length %d, rune length %d)\n"+
					"actual:   '%s' (byte length %d, rune length %d)\n"+hint,
					testCase.result, len(testCase.result), len(expectedRunes),
					actual, len(actual), len(actualRunes))
				t.Fail()
			}
		})
	}
}

func TestParseMastodonHandle(t *testing.T) {
	t.Run("user@server", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		user, server, err := ParseMastodonHandle("user@server")
		require.NoError(err)
		assert.Equal("user", user)
		assert.Equal("server", server)
	})

	t.Run("@user@server", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		user, server, err := ParseMastodonHandle("@user@server")
		require.NoError(err)
		assert.Equal("user", user)
		assert.Equal("server", server)
	})
}
