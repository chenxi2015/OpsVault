package credutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	// Exclude ambiguous characters like 0/O, 1/l/I for readability
	passwordLower   = "abcdefghjkmnpqrstuvwxyz"
	passwordUpper   = "ABCDEFGHJKMNPQRSTUVWXYZ"
	passwordDigits  = "23456789"
	passwordSpecial = "!@#$%^&*"
)

// GenPassword generates a cryptographically random password of the given length.
// The result always contains at least one character from each character class.
func GenPassword(length int) string {
	if length < 8 {
		length = 8
	}
	all := passwordLower + passwordUpper + passwordDigits + passwordSpecial
	classes := []string{passwordLower, passwordUpper, passwordDigits, passwordSpecial}

	buf := make([]byte, length)
	// Ensure at least one character from each class
	for i, class := range classes {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(class))))
		buf[i] = class[idx.Int64()]
	}
	// Fill the rest randomly from all chars
	for i := len(classes); i < length; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		buf[i] = all[idx.Int64()]
	}
	// Shuffle buffer using Fisher-Yates with crypto/rand
	for i := length - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		buf[i], buf[j.Int64()] = buf[j.Int64()], buf[i]
	}
	return string(buf)
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

