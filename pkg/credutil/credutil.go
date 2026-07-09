package credutil

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sethvargo/go-password/password"
)

var defaultGenerator *password.Generator

func init() {
	var err error
	defaultGenerator, err = password.NewGenerator(&password.GeneratorInput{
		Digits:       "23456789",                // Exclude ambiguous 0, 1
		Symbols:      "!@$%^&*",                 // Exclude #, ", ', \, space to avoid config parsing issues
		LowerLetters: "abcdefghjkmnpqrstuvwxyz", // Exclude ambiguous l, o
		UpperLetters: "ABCDEFGHJKMNPQRSTUVWXYZ", // Exclude ambiguous I, O
	})
	if err != nil {
		defaultGenerator = nil
	}
}

// GenPassword generates a cryptographically random password of the given length.
// The result always contains at least one character from each character class.
func GenPassword(length int) string {
	if length < 8 {
		length = 8
	}
	if defaultGenerator != nil {
		pwd, err := defaultGenerator.Generate(length, length/4, length/4, false, false)
		if err == nil {
			return pwd
		}
	}
	pwd, _ := password.Generate(length, length/4, length/4, false, false)
	return pwd
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
