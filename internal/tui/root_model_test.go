package tui

import (
	"strings"
	"testing"
	"time"

	"OpsVault/internal/driver"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRootModelTickRefreshesStatuses(t *testing.T) {
	model := NewRootModel(StaticStatusProvider{
		"nginx":    {Name: "nginx", Mode: driver.ModeBinary, Status: "running", Running: true},
		"mysql":    {Name: "mysql", Mode: driver.ModeDocker, Status: "healthy", Running: true},
		"rocketmq": {Name: "rocketmq", Mode: driver.ModeDocker, Status: "starting", Running: false},
	})

	msg := statusesLoadedMsg{
		services: []driver.ServiceStatus{
			{Name: "nginx", Mode: driver.ModeBinary, Status: "running", Running: true},
			{Name: "mysql", Mode: driver.ModeDocker, Status: "healthy", Running: true},
			{Name: "rocketmq", Mode: driver.ModeDocker, Status: "starting", Running: false},
		},
	}

	updated, cmd := model.Update(msg)
	root := updated.(RootModel)
	if cmd == nil {
		t.Fatal("expected next refresh command")
	}
	if len(root.services) != 3 {
		t.Fatalf("services len = %d, want 3", len(root.services))
	}
	view := root.View()
	for _, want := range []string{"nginx", "mysql", "rocketmq", "healthy"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestRootModelTickCommandEmitsRefreshMessage(t *testing.T) {
	model := NewRootModel(StaticStatusProvider{
		"nginx": {Name: "nginx", Status: "running", Running: true},
	})

	cmd := model.Init()
	if cmd == nil {
		t.Fatal("Init returned nil command")
	}
	msg := cmd()
	switch actual := msg.(type) {
	case tea.BatchMsg:
		if len(actual) == 0 {
			t.Fatal("batch message is empty")
		}
	case refreshTickMsg:
	default:
		t.Fatalf("unexpected init message type %T", msg)
	}
}

func TestLoadStatusesReturnsRefreshMessage(t *testing.T) {
	model := NewRootModel(StaticStatusProvider{
		"nginx": {Name: "nginx", Status: "running", Running: true, UpdatedAt: time.Now()},
		"mysql": {Name: "mysql", Status: "healthy", Running: true, UpdatedAt: time.Now()},
	})

	msg := model.loadStatuses()()
	loaded, ok := msg.(statusesLoadedMsg)
	if !ok {
		t.Fatalf("message type = %T, want statusesLoadedMsg", msg)
	}
	if len(loaded.services) != 2 {
		t.Fatalf("services len = %d, want 2", len(loaded.services))
	}
}
