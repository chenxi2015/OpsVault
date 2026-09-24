package credutil

import (
	"strings"
	"testing"
)

func TestSafePasswordCharsetExcludesProblemChars(t *testing.T) {
	for _, ch := range safePasswordCharset {
		if strings.ContainsRune("@#$%&*\\/:;'\"`", ch) {
			t.Fatalf("unsafe password charset contains config-problem character %q", ch)
		}
	}
}

func TestGenPasswordAvoidsUnsafeCharacters(t *testing.T) {
	for i := 0; i < 200; i++ {
		password := GenPassword(20)
		for _, ch := range password {
			if strings.ContainsRune("@#$%&*\\/:;'\"`", ch) {
				t.Fatalf("generated password contains unsafe character %q: %q", ch, password)
			}
		}
	}
}
