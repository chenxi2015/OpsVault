package tui

import (
	"strings"
	"testing"
	"time"

	"OpsVault/internal/driver"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
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

	updated, cmd := (&model).Update(msg)
	root := updated.(*RootModel)
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

	cmd := (&model).Init()
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

	msg := (&model).loadStatuses()()
	loaded, ok := msg.(statusesLoadedMsg)
	if !ok {
		t.Fatalf("message type = %T, want statusesLoadedMsg", msg)
	}
	if len(loaded.services) != 2 {
		t.Fatalf("services len = %d, want 2", len(loaded.services))
	}
}

type mockNginxDriver struct {
	driver.ServiceDriver
}

func (mockNginxDriver) TailLogs(lines int) (string, error) {
	return "mock logs", nil
}

func (mockNginxDriver) Reload() error {
	return nil
}

func TestServiceRegistryAndActions(t *testing.T) {
	// Assert presence of registry types and actions
	ref := ServiceRef{
		Name:   "nginx",
		Driver: mockNginxDriver{},
	}
	if ref.Name != "nginx" {
		t.Errorf("expected name nginx, got %s", ref.Name)
	}

	// Mock service status
	status := driver.ServiceStatus{
		Name:    "nginx",
		Mode:    driver.ModeBinary,
		Running: true,
		Status:  "running",
	}

	actions := AvailableServiceActions(status, ref)
	hasLogs := false
	for _, a := range actions {
		if a.ID == ActionLogs {
			hasLogs = true
		}
	}
	if !hasLogs {
		t.Errorf("running Nginx service should have 'logs' action available")
	}
}

func TestRootModelFocusAndDrawer(t *testing.T) {
	model := NewRootModel(StaticStatusProvider{})
	// Confirm focus starts at sidebar or detail
	if model.focus != focusSidebar {
		t.Errorf("expected initial focus to be focusSidebar, got %v", model.focus)
	}

	// Simulate tab key to cycle focus
	updated, _ := (&model).Update(tea.KeyMsg{Type: tea.KeyTab, Runes: []rune{}})
	root := updated.(*RootModel)
	if root.focus != focusDetail {
		t.Errorf("expected Tab key to cycle focus to focusDetail, got %v", root.focus)
	}

	// Confirm drawer mode defaults to hidden
	if root.drawerMode != drawerHidden {
		t.Errorf("expected initial drawer to be drawerHidden, got %v", root.drawerMode)
	}
}

func TestRootModelConfigTab(t *testing.T) {
	model := NewRootModel(StaticStatusProvider{})
	model.active = 4 // Config Tab
	model.focus = focusSidebar

	// Verify Category selection moves down
	updated, _ := (&model).Update(tea.KeyMsg{Type: tea.KeyDown, Runes: []rune{}})
	root := updated.(*RootModel)
	if root.selectedConfigCategory != 1 {
		t.Errorf("expected selectedConfigCategory to be 1, got %d", root.selectedConfigCategory)
	}

	// Verify Enter moves focus to details
	updated, _ = root.Update(tea.KeyMsg{Type: tea.KeyEnter, Runes: []rune{}})
	root = updated.(*RootModel)
	if root.focus != focusDetail {
		t.Errorf("expected enter on sidebar to change focus to focusDetail, got %v", root.focus)
	}
	if root.selectedConfigItem != 0 {
		t.Errorf("expected selectedConfigItem to be reset to 0, got %d", root.selectedConfigItem)
	}

	// Verify Item selection moves down
	updated, _ = root.Update(tea.KeyMsg{Type: tea.KeyDown, Runes: []rune{}})
	root = updated.(*RootModel)
	if root.selectedConfigItem != 1 {
		t.Errorf("expected selectedConfigItem to be 1, got %d", root.selectedConfigItem)
	}

	// Verify Esc moves focus back to sidebar
	updated, _ = root.Update(tea.KeyMsg{Type: tea.KeyEsc, Runes: []rune{}})
	root = updated.(*RootModel)
	if root.focus != focusSidebar {
		t.Errorf("expected Esc on details to change focus back to focusSidebar, got %v", root.focus)
	}
}

func TestGitLabConfigKeysAndInput(t *testing.T) {
	var gitlabCat *ConfigCategory
	for i := range ConfigCategories {
		if ConfigCategories[i].Name == "GitLab" {
			gitlabCat = &ConfigCategories[i]
			break
		}
	}
	if gitlabCat == nil {
		t.Fatalf("GitLab category not found in ConfigCategories")
	}

	hasPumaWorkers := false
	hasExternalURL := false
	for _, k := range gitlabCat.Keys {
		if k == "gitlab.puma_workers" {
			hasPumaWorkers = true
		}
		if k == "gitlab.external_url" {
			hasExternalURL = true
		}
	}
	if !hasPumaWorkers {
		t.Errorf("expected gitlab.puma_workers in GitLab category keys")
	}
	if !hasExternalURL {
		t.Errorf("expected gitlab.external_url in GitLab category keys")
	}

	model := NewRootModel(StaticStatusProvider{})
	model.config = viper.New()
	model.textInputState = "config|gitlab.puma_workers"
	model.textInputValue = "4"
	model.handleInputSubmit()

	if model.config.GetInt("gitlab.puma_workers") != 4 {
		t.Errorf("expected gitlab.puma_workers to be 4, got %v", model.config.Get("gitlab.puma_workers"))
	}
}

func TestConfigCategoriesSystemSettingsFirst(t *testing.T) {
	if len(ConfigCategories) == 0 {
		t.Fatal("expected ConfigCategories to have entries")
	}
	firstCat := ConfigCategories[0]
	if firstCat.Name != "System Settings" {
		t.Errorf("expected first category to be 'System Settings', got %q", firstCat.Name)
	}

	hasRootDir := false
	for _, key := range firstCat.Keys {
		if key == "system.root_dir" {
			hasRootDir = true
			break
		}
	}
	if !hasRootDir {
		t.Errorf("expected 'system.root_dir' in System Settings keys, got %v", firstCat.Keys)
	}
}

func TestConfigWizardViewWithDefaultYaml(t *testing.T) {
	v := viper.New()
	v.Set("system.root_dir", "/www/opsvault")
	v.Set("nginx.www_root", "/www/opsvault/wwwroot")
	v.Set("nginx.ssl_root", "/www/opsvault/ssl")
	v.Set("nginx.wwwlogs_root", "/www/opsvault/wwwlogs")
	v.Set("docker.data_root", "/www/opsvault")

	model := NewRootModel(StaticStatusProvider{})
	model.config = v
	model.selectedConfigCategory = 0 // System Settings

	viewSys := ConfigWizardView(model)
	if !strings.Contains(viewSys, "SYSTEM SETTINGS CONFIGURATIONS") {
		t.Errorf("expected view to contain 'SYSTEM SETTINGS CONFIGURATIONS'")
	}
	if !strings.Contains(viewSys, "system.root_dir") || !strings.Contains(viewSys, "/www/opsvault") {
		t.Errorf("expected view to display system.root_dir and /www/opsvault")
	}

	// Switch to Nginx category
	for idx, cat := range ConfigCategories {
		if cat.Name == "Nginx" {
			model.selectedConfigCategory = idx
			break
		}
	}
	viewNginx := ConfigWizardView(model)
	if !strings.Contains(viewNginx, "/www/opsvault/wwwroot") {
		t.Errorf("expected nginx.www_root to display /www/opsvault/wwwroot")
	}
	if !strings.Contains(viewNginx, "/www/opsvault/wwwlogs") {
		t.Errorf("expected nginx.wwwlogs_root to display /www/opsvault/wwwlogs")
	}
	if !strings.Contains(viewNginx, "/www/opsvault/ssl") {
		t.Errorf("expected nginx.ssl_root to display /www/opsvault/ssl")
	}
}


