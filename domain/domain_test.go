package domain

import (
	"strings"
	"testing"
)

func TestProjectIdentityValidation(t *testing.T) {
	project := Project{ID: ProjectID("project-1"), Name: "Demo", Path: "/tmp/demo"}
	if err := project.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Project{ID: "", Name: "Demo"}).Validate(); err == nil {
		t.Fatal("empty project ID was accepted")
	}
	if err := (Project{ID: "project-1", Name: "bad\nname"}).Validate(); err == nil {
		t.Fatal("control character in project name was accepted")
	}
}

func TestSecretNameValidation(t *testing.T) {
	valid := []SecretName{"OPENAI_API_KEY", "服务令牌", "name with spaces"}
	for _, name := range valid {
		if err := name.Validate(); err != nil {
			t.Fatalf("valid name %q rejected: %v", name, err)
		}
	}
	for _, name := range []SecretName{"", " ", "bad=name", "bad\nname", SecretName(strings.Repeat("x", 256))} {
		if err := name.Validate(); err == nil {
			t.Fatalf("invalid name %q was accepted", name)
		}
	}
}

func TestSecretValueIsNotPartOfMetadata(t *testing.T) {
	secret := Secret{ProjectID: "project-1", Name: "TOKEN"}
	if err := secret.Validate(); err != nil {
		t.Fatal(err)
	}
	material := SecretMaterial{Name: "TOKEN", Value: "secret-value"}
	if got := material.String(); strings.Contains(got, material.Value) {
		t.Fatalf("secret material String leaked value: %q", got)
	}
}

func TestExecutionRequestValidation(t *testing.T) {
	request := ExecutionRequest{
		ProjectID:   "project-1",
		Command:     []string{"npm", "run", "dev"},
		SecretNames: []SecretName{"OPENAI_API_KEY"},
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []ExecutionRequest{
		{ProjectID: "project-1"},
		{ProjectID: "project-1", Command: []string{"bad\x00command"}},
		{ProjectID: "project-1", Command: []string{"app"}, SecretNames: []SecretName{"TOKEN", "TOKEN"}},
	} {
		if err := invalid.Validate(); err == nil {
			t.Fatalf("invalid request was accepted: %#v", invalid)
		}
	}
}

func TestScanFindingValidation(t *testing.T) {
	if err := (ScanFinding{Severity: SeverityWarning, Path: "main.go", Line: 12, Column: 3, Category: "credential", Message: "possible plaintext credential"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (ScanFinding{Severity: "unknown", Message: "finding"}).Validate(); err == nil {
		t.Fatal("unknown scan severity was accepted")
	}
}

func FuzzDomainValidation(f *testing.F) {
	f.Add("project-1", "TOKEN_NAME", "secret value", "path/to/file")
	f.Add("", " ", "val\x00bad", "bad\npath")
	f.Add("proj", "KEY=VALUE", "val", "file.go")

	f.Fuzz(func(t *testing.T, proj, name, val, path string) {
		_ = (ProjectID(proj)).Validate()
		_ = (Project{ID: ProjectID(proj), Name: proj, Path: path}).Validate()
		_ = (SecretName(name)).Validate()
		_ = (SecretInput{ProjectID: ProjectID(proj), Name: SecretName(name), Value: val}).Validate()
		_ = (ScanFinding{Severity: SeverityWarning, Path: path, Category: "test", Message: val}).Validate()
	})
}
