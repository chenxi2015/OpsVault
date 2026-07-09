package cmd

import (
	"testing"
)

func TestNewInitModelIncludesJenkinsAndGitLab(t *testing.T) {
	m := newInitModel()
	if m == nil {
		t.Fatal("expected non-nil initModel")
	}

	hasJenkins := false
	hasGitLab := false
	for _, item := range m.items {
		if item.name == "jenkins" {
			hasJenkins = true
		}
		if item.name == "gitlab" {
			hasGitLab = true
		}
	}

	if !hasJenkins {
		t.Error("expected initModel to include 'jenkins'")
	}
	if !hasGitLab {
		t.Error("expected initModel to include 'gitlab'")
	}
}
