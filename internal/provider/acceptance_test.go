package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/harryvince/terraform-provider-cups/internal/cups"
)

// This test intentionally runs the real Terraform CLI with a locally built
// provider. It never starts or targets a server without an explicit opt-in.
func TestAcceptancePrinter(t *testing.T) {
	if os.Getenv("CUPS_ACC") != "1" {
		t.Skip("set CUPS_ACC=1 and CUPS_ACC_ENDPOINT/USERNAME/PASSWORD for the isolated fixture")
	}
	endpoint := os.Getenv("CUPS_ACC_ENDPOINT")
	u, err := url.Parse(endpoint)
	if err != nil || u == nil || u.Hostname() == "" || u.Port() == "" || u.Port() == "631" {
		t.Fatal("CUPS_ACC_ENDPOINT must explicitly target a loopback test port other than 631")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
		t.Fatal("acceptance tests require a loopback endpoint")
	}
	username, password := os.Getenv("CUPS_ACC_USERNAME"), os.Getenv("CUPS_ACC_PASSWORD")
	client, err := cups.New(endpoint, username, password, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	terraform, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("Terraform CLI is required for acceptance tests")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(bin, "terraform-provider-cups"), ".")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("provider build: %s\n%s", err, output)
	}
	rc := filepath.Join(dir, "terraformrc")
	if err := os.WriteFile(rc, []byte(fmt.Sprintf("provider_installation {\n dev_overrides {\n  \"registry.terraform.io/harryvince/cups\" = %s\n }\n direct {}\n}\n", strconv.Quote(bin))), 0600); err != nil {
		t.Fatal(err)
	}
	var environment []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "TF_") && !strings.HasPrefix(entry, "CUPS_") {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "TF_CLI_CONFIG_FILE="+rc, "TF_IN_AUTOMATION=1", "TF_INPUT=0", "CHECKPOINT_DISABLE=1",
		"CUPS_ENDPOINT="+endpoint, "CUPS_USERNAME="+username, "CUPS_PASSWORD="+password)
	run := func(expected int, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, terraform, args...)
		cmd.Dir, cmd.Env = dir, environment
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		output, err := cmd.Output()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("terraform %s: %v", args[0], err)
			}
			code = exit.ExitCode()
		}
		if code != expected {
			t.Fatalf("terraform %s: exit %d, expected %d\n%s", strings.Join(args, " "), code, expected, strings.ReplaceAll(string(output)+stderr.String(), password, "[redacted]"))
		}
		if args[0] == "show" {
			return string(output)
		}
		return string(output) + stderr.String()
	}
	name := fmt.Sprintf("acc-%d", time.Now().UnixNano())
	renamed, duplicate := name+"-new", name+"-existing"
	t.Cleanup(func() {
		for _, queue := range []string{name, renamed, duplicate} {
			if err := client.Delete(context.Background(), queue); err != nil {
				t.Errorf("cleanup queue %s: %v", queue, err)
			}
		}
	})
	writeConfig := func(queue, description, location string, includeMetadata bool) {
		t.Helper()
		metadata := ""
		if includeMetadata {
			metadata = fmt.Sprintf("description = %s\nlocation = %s\n", strconv.Quote(description), strconv.Quote(location))
		}
		config := fmt.Sprintf(`terraform {
  required_providers {
    cups = { source = "registry.terraform.io/harryvince/cups" }
  }
}
provider "cups" {}
resource "cups_printer" "test" {
  name = %s
  device_uri = "ipp://printer.local:8000/ipp/print"
  %s
}
`, strconv.Quote(queue), metadata)
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(config), 0600); err != nil {
			t.Fatal(err)
		}
	}
	assertQueue := func(queue, description, location string) {
		t.Helper()
		p, err := client.Read(context.Background(), queue)
		if err != nil {
			t.Fatal(err)
		}
		if p.Description != description || p.Location != location {
			t.Fatalf("CUPS configuration differs: %#v", p)
		}
		var state struct {
			Values struct {
				RootModule struct {
					Resources []struct{ Values map[string]any }
				} `json:"root_module"`
			}
		}
		if err := json.Unmarshal([]byte(run(0, "show", "-json")), &state); err != nil {
			t.Fatal(err)
		}
		if len(state.Values.RootModule.Resources) != 1 {
			t.Fatal("expected one resource in state")
		}
		values := state.Values.RootModule.Resources[0].Values
		for key, expected := range map[string]string{"id": queue, "name": queue, "description": description, "location": location, "device_uri": "ipp://printer.local:8000/ipp/print"} {
			if values[key] != expected {
				t.Fatalf("state %s = %v, expected %q", key, values[key], expected)
			}
		}
	}

	writeConfig(name, "Acceptance printer", "Room A", true)
	run(0, "validate", "-no-color")
	run(0, "apply", "-auto-approve", "-no-color")
	assertQueue(name, "Acceptance printer", "Room A")
	run(0, "plan", "-detailed-exitcode", "-no-color")
	t.Log("create/read and stable plan passed")
	// A device change must regenerate the model through replacement, not reuse an
	// existing PPD. Planning an unreachable device must not contact that device.
	configPath := filepath.Join(dir, "main.tf")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(strings.ReplaceAll(string(config), ":8000/", ":8001/")), 0600); err != nil {
		t.Fatal(err)
	}
	if output := run(2, "plan", "-detailed-exitcode", "-no-color"); !strings.Contains(output, "forces replacement") {
		t.Fatal("device URI change did not require replacement")
	}
	if err := os.WriteFile(configPath, config, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("device URI change plans replacement without probing the new device")

	writeConfig(name, "Updated printer", "Room B", true)
	run(0, "apply", "-auto-approve", "-no-color")
	assertQueue(name, "Updated printer", "Room B")
	run(0, "plan", "-detailed-exitcode", "-no-color")
	t.Log("update and stable plan passed")

	run(0, "state", "rm", "cups_printer.test")
	run(0, "import", "-no-color", "cups_printer.test", name)
	assertQueue(name, "Updated printer", "Room B")
	run(0, "plan", "-detailed-exitcode", "-no-color")
	t.Log("import passed")

	p, err := client.Read(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}
	p.Location = "External drift"
	if _, err := client.Update(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	run(2, "plan", "-detailed-exitcode", "-no-color")
	run(0, "apply", "-auto-approve", "-no-color")
	assertQueue(name, "Updated printer", "Room B")
	t.Log("external drift detection and correction passed")

	writeConfig(name, "", "", false)
	run(0, "apply", "-auto-approve", "-no-color")
	assertQueue(name, "", "")
	run(0, "plan", "-detailed-exitcode", "-no-color")
	t.Log("omitted metadata clears attributes without perpetual diffs")

	writeConfig(renamed, "", "", false)
	output := run(2, "plan", "-detailed-exitcode", "-no-color")
	if !strings.Contains(output, "forces replacement") {
		t.Fatal("name change did not require replacement")
	}
	run(0, "apply", "-auto-approve", "-no-color")
	assertQueue(renamed, "", "")
	if _, err := client.Read(context.Background(), name); !errors.Is(err, cups.ErrNotFound) {
		t.Fatalf("old queue was not deleted: %v", err)
	}
	t.Log("name replacement passed")

	if err := client.Delete(context.Background(), renamed); err != nil {
		t.Fatal(err)
	}
	run(2, "plan", "-detailed-exitcode", "-no-color")
	run(0, "apply", "-auto-approve", "-no-color")
	assertQueue(renamed, "", "")
	t.Log("external deletion and recreation passed")
	run(0, "destroy", "-auto-approve", "-no-color")
	if _, err := client.Read(context.Background(), renamed); !errors.Is(err, cups.ErrNotFound) {
		t.Fatalf("queue survived destroy: %v", err)
	}

	existing := cups.Printer{Name: duplicate, DeviceURI: "ipp://printer.local:8000/ipp/print", Description: "Existing printer", Location: "Existing room"}
	if _, err := client.Create(context.Background(), existing); err != nil {
		t.Fatal(err)
	}
	writeConfig(duplicate, "Overwritten", "Overwritten", true)
	output = run(1, "apply", "-auto-approve", "-no-color")
	if !strings.Contains(strings.Join(strings.Fields(output), " "), "import it instead") {
		t.Fatalf("duplicate creation did not explain import: %s", output)
	}
	actual, err := client.Read(context.Background(), duplicate)
	if err != nil || actual != existing {
		t.Fatalf("duplicate create modified existing queue: %#v, %v", actual, err)
	}
	writeConfig(duplicate, existing.Description, existing.Location, true)
	run(0, "import", "-no-color", "cups_printer.test", duplicate)
	run(0, "plan", "-detailed-exitcode", "-no-color")
	run(0, "destroy", "-auto-approve", "-no-color")
	t.Log("duplicate protection, import of existing queue, and destroy passed")
}
