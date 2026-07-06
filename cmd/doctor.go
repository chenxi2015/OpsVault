package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"OpsVault/internal/system"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Lipgloss styles for CLI beauty optimization
var (
	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")). // Light Blue
			Border(lipgloss.DoubleBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("39")).
			PaddingBottom(1).
			MarginBottom(1)

	// Status badges with colored backgrounds
	passBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("10")). // Light Green
			Padding(0, 1)

	warnBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("11")). // Light Yellow
			Padding(0, 1)

	failBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("9")). // Light Red
			Padding(0, 1)

	// Left border style boxes for clean visual hierarchy
	cardBaseStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(1)

	passCard = cardBaseStyle.Copy().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderLeftForeground(lipgloss.Color("10"))

	warnCard = cardBaseStyle.Copy().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderLeftForeground(lipgloss.Color("11"))

	failCard = cardBaseStyle.Copy().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderLeftForeground(lipgloss.Color("9"))

	// Typography styles
	itemNameStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	messageStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	suggestionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // Muted gray

	// Summary box style
	summaryStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 3).
			MarginTop(1).
			MarginBottom(1)

	successBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("10")).
			Padding(0, 2)

	warnBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("11")).
			Padding(0, 2)

	failBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("9")).
			Padding(0, 2)
)

func newDoctorCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "诊断本地/服务器运维环境 (Lipgloss 风格)",
		Long:  "对操作系统、用户权限、持久化存储根目录、Docker 连通性、专属网桥、端口占用、编译依赖等进行全面环境体检。",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Title banner
			cmd.Println(bannerStyle.Render("OpsVault Environment Doctor - 运行环境体检诊断"))

			// Initialize Docker Client
			dockerCli, _ := dockerFactory()

			// Run diagnostics
			ctx := context.Background()
			items, err := system.RunDiagnostics(ctx, cfg, dockerCli)
			if err != nil {
				return fmt.Errorf("环境诊断时发生异常错误: %w", err)
			}

			var passed, warnings, failures int

			for _, item := range items {
				var badgeStr string
				var cardStyle lipgloss.Style

				switch item.Status {
				case system.StatusOk:
					badgeStr = passBadge.Render(" PASS ")
					cardStyle = passCard
					passed++
				case system.StatusWarn:
					badgeStr = warnBadge.Render(" WARN ")
					cardStyle = warnCard
					warnings++
				case system.StatusFail:
					badgeStr = failBadge.Render(" FAIL ")
					cardStyle = failCard
					failures++
				}

				// Build item detail content
				titleLine := fmt.Sprintf("%s  %s  %s", badgeStr, itemNameStyle.Render(item.Name), messageStyle.Render(item.Message))
				var contentLines []string
				contentLines = append(contentLines, titleLine)

				if item.Suggestion != "" && item.Status != system.StatusOk {
					contentLines = append(contentLines, fmt.Sprintf("💡 修复建议: %s", suggestionStyle.Render(item.Suggestion)))
				}

				// Render styled card
				cmd.Println(cardStyle.Render(strings.Join(contentLines, "\n")))
			}

			// Render summary card
			total := len(items)
			summaryText := fmt.Sprintf(
				"环境诊断体检报告汇总:\n\n"+
					"  ● 总检查数  : %d 项\n"+
					"  ✔ 通过项    : %d 项\n"+
					"  ⚠️ 警告项    : %d 项\n"+
					"  ✘ 失败项    : %d 项",
				total, passed, warnings, failures,
			)
			cmd.Println(summaryStyle.Render(summaryText))

			// Exit code decision and colored status flags
			if failures > 0 {
				cmd.Println(failBanner.Render(fmt.Sprintf(" ✗ 环境诊断未通过，检测到 %d 个关键故障！请优先处理红牌 [FAIL] 项。 ", failures)))
				cmd.Println()
				os.Exit(1)
			} else if warnings > 0 {
				cmd.Println(warnBanner.Render(fmt.Sprintf(" ⚠️ 环境诊断通过，但存在 %d 个警告，请确认这是否会影响您的实际部署需求。 ", warnings)))
				cmd.Println()
			} else {
				cmd.Println(successBanner.Render(" ✔ 恭喜！所有环境检查项全部通过，您的服务器非常健康，可以正常部署服务。 "))
				cmd.Println()
			}

			return nil
		},
	}
}
