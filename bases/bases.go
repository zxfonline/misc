// Utility for converting numbers to/from different bases/alphabets.
package bases

import (
	"math"
	"regexp"
	"strings"
)

// Known alphabets:
const (
	NUMERALS          = "0123456789"
	LETTERS_LOWERCASE = "abcdefghijklmnopqrstuvwxyz"
)

var (
	/*
		2=01
		3=012
		4=0123
		5=01234
		6=012345
		7=0123456
		8=01234567
		9=012345678
		10=0123456789
		11=0123456789a
		12=0123456789ab
		13=0123456789abc
		14=0123456789abcd
		15=0123456789abcde
		16=0123456789abcdef
		26=abcdefghijklmnopqrstuvwxyz
		32=0123456789ABCDEFGHJKMNPQRSTVWXYZ
		36=0123456789abcdefghijklmnopqrstuvwxyz
		52=abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
		58=123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ
		62=0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
		64=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/
	*/
	KNOWN_ALPHABETS   = make(map[int]string)
	LETTERS_UPPERCASE = strings.ToUpper(LETTERS_LOWERCASE)
)

func init() {
	// Each of the number ones, starting from base-2 (base-1 doesn't make sense?):
	for i := 2; i <= 10; i++ {
		KNOWN_ALPHABETS[i] = NUMERALS[0:i]
	}

	// Node's native hex is 0-9 followed by *lowercase* a-f, so we'll take that
	// approach for everything from base-11 to base-16:
	for i := 11; i <= 16; i++ {
		KNOWN_ALPHABETS[i] = NUMERALS + LETTERS_LOWERCASE[0:i-10]
	}

	// We also model base-36 off of that, just using the full letter alphabet:
	KNOWN_ALPHABETS[36] = NUMERALS + LETTERS_LOWERCASE

	// And base-62 will be the uppercase letters added:
	KNOWN_ALPHABETS[62] = NUMERALS + LETTERS_LOWERCASE + LETTERS_UPPERCASE

	// For base-26, we'll assume the user wants just the letter alphabet:
	KNOWN_ALPHABETS[26] = LETTERS_LOWERCASE

	// We'll also add a similar base-52, just letters, lowercase then uppercase:
	KNOWN_ALPHABETS[52] = LETTERS_LOWERCASE + LETTERS_UPPERCASE

	// Base-64 is a formally-specified alphabet that has a particular order:
	// http://en.wikipedia.org/wiki/Base64 (and Node.js follows this too)
	// TODO FIXME But our code above doesn't add padding! Don't use this yet...
	KNOWN_ALPHABETS[64] = LETTERS_UPPERCASE + LETTERS_LOWERCASE + NUMERALS + "+/"

	// Flickr and others also have a base-58 that removes confusing characters, but
	// there isn't consensus on the order of lowercase vs. uppercase... =/
	// http://www.flickr.com/groups/api/discuss/72157616713786392/
	// https://en.bitcoin.it/wiki/Base58Check_encoding#Base58_symbol_chart
	// https://github.com/dougal/base58/blob/master/lib/base58.rb
	// http://icoloma.blogspot.com/2010/03/create-your-own-bitly-using-base58.html
	// We'll arbitrarily stay consistent with the above and using lowercase first:
	KNOWN_ALPHABETS[58] = regexp.MustCompile("[0OlI]").ReplaceAllString(KNOWN_ALPHABETS[62], "")

	// And Douglas Crockford shared a similar base-32 from base-36:
	// http://www.crockford.com/wrmg/base32.html
	// Unlike our base-36, he explicitly specifies uppercase letters
	KNOWN_ALPHABETS[32] = NUMERALS + regexp.MustCompile("[ILOU]").ReplaceAllString(LETTERS_UPPERCASE, "")
}

// Returns a string representation of the given number for the given alphabet:
func toAlphabet(num int, alphabet string) string {
	base := len(alphabet)
	var digits []int // these will be in reverse order since arrays are stacks

	// execute at least once, even if num is 0, since we should return the '0':
	for {
		digits = append(digits, num%base) // TODO handle negatives properly?
		num /= base
		if num <= 0 {
			break
		}
	}
	var chars []byte
	for len(digits) > 0 {
		n := len(digits)
		digit := digits[n-1]
		digits = digits[0 : n-1]
		chars = append(chars, alphabet[digit])
	}
	return string(chars)
}

// Returns an integer representation of the given string for the given alphabet:
func fromAlphabet(str string, alphabet string) int {
	base := len(alphabet)
	pos := 0
	num := 0
	var c byte
	for len(str) > 0 {
		n := len(str)
		c = str[n-1]
		str = str[0 : n-1]
		num += int(math.Pow(float64(base), float64(pos)) * float64(strings.IndexByte(alphabet, c)))
		pos++
	}
	return num
}

// And a generic alias too:
func ToBase(num int, base int) string {
	return toAlphabet(num, KNOWN_ALPHABETS[base])
}

func FromBase(str string, base int) int {
	return fromAlphabet(str, KNOWN_ALPHABETS[base])
}
