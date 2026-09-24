package credutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const safePasswordCharset = "ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!-_+.?"

// GenPassword generates a cryptographically random password with a balanced charset.
// It keeps a few common safe punctuation marks to improve entropy while avoiding config-breaking
// characters such as @, $, #, &, *, \, /, :, ;, quotes, backticks, and spaces.
func GenPassword(length int) string {
	if length < 12 {
		length = 12
	}

	lowerLetters := "abcdefghjkmnpqrstuvwxyz"
	upperLetters := "ABCDEFGHJKMNPQRSTUVWXYZ"
	digits := "23456789"
	safeSymbols := "!-_+.?"

	chars := make([]byte, length)
	for i := 0; i < length; i++ {
		chars[i] = safePasswordCharset[randomIndex(len(safePasswordCharset))]
	}

	chars[0] = lowerLetters[randomIndex(len(lowerLetters))]
	chars[1] = upperLetters[randomIndex(len(upperLetters))]
	chars[2] = digits[randomIndex(len(digits))]
	chars[3] = safeSymbols[randomIndex(len(safeSymbols))]

	for i := range chars {
		if chars[i] == 0 {
			chars[i] = safePasswordCharset[randomIndex(len(safePasswordCharset))]
		}
	}

	return string(chars)
}

func randomIndex(max int) int {
	if max <= 0 {
		return 0
	}
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(idx.Int64())
}

// Credential holds a single key-value pair for display.
type Credential struct {
	Label string
	Value string
}

// RenderCredentials returns a styled credential card as a string.
func RenderCredentials(serviceName string, creds []Credential) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11"))

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	valStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10"))

	warnStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		Padding(1, 3)

	var lines []string
	lines = append(lines, titleStyle.Render(fmt.Sprintf("🔐  %s 服务凭据 — 请妥善保管！", serviceName)))
	lines = append(lines, strings.Repeat("─", 42))
	lines = append(lines, "")

	// Calculate max label length for alignment
	maxLabelLen := 0
	for _, c := range creds {
		if len([]rune(c.Label)) > maxLabelLen {
			maxLabelLen = len([]rune(c.Label))
		}
	}
	for _, c := range creds {
		pad := strings.Repeat(" ", maxLabelLen-len([]rune(c.Label)))
		lines = append(lines, fmt.Sprintf("%s  %s",
			keyStyle.Render(c.Label+pad+" :"),
			valStyle.Render(c.Value),
		))
	}

	lines = append(lines, "")
	lines = append(lines, warnStyle.Render("⚠️  此密码不会再次显示，请立即记录！"))

	return boxStyle.Render(strings.Join(lines, "\n"))
}

// PrintCredentials prints a styled credential card to stdout after successful installation.
// serviceName is the display name (e.g. "MySQL"), creds is an ordered list of labels+values.
func PrintCredentials(serviceName string, creds []Credential) {
	fmt.Println()
	fmt.Println(RenderCredentials(serviceName, creds))
	fmt.Println()
}
