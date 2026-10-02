package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func dynamicConfig(t *testing.T, typ tftypes.Type, values map[string]any) *tfprotov6.DynamicValue {
	t.Helper()
	object := typ.(tftypes.Object)
	attrs := make(map[string]tftypes.Value)
	for name, attrType := range object.AttributeTypes {
		attrs[name] = tftypes.NewValue(attrType, values[name])
	}
	config, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, attrs))
	if err != nil {
		t.Fatal(err)
	}
	return &config
}

func hasError(diagnostics []*tfprotov6.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

func TestProviderConfiguration(t *testing.T) {
	t.Setenv("CUPS_ENDPOINT", "http://127.0.0.1:8631")
	t.Setenv("CUPS_USERNAME", "cups-admin")
	t.Setenv("CUPS_PASSWORD", "fixture-secret")
	for _, tc := range []struct {
		name      string
		values    map[string]any
		wantError bool
	}{
		{"environment fallback", map[string]any{}, false},
		{"explicit values override environment", map[string]any{"endpoint": "https://cups.example.test:631", "username": "explicit-admin", "password": "explicit-secret"}, false},
		{"empty endpoint overrides environment", map[string]any{"endpoint": ""}, true},
		{"empty password overrides environment", map[string]any{"password": ""}, true},
		{"unknown endpoint does not fall back", map[string]any{"endpoint": tftypes.UnknownValue}, true},
		{"unknown password does not fall back", map[string]any{"password": tftypes.UnknownValue}, true},
		{"unknown timeout", map[string]any{"request_timeout": tftypes.UnknownValue}, true},
		{"zero timeout", map[string]any{"request_timeout": int64(0)}, true},
		{"large timeout", map[string]any{"request_timeout": int64(301)}, true},
		{"valid timeout", map[string]any{"request_timeout": int64(60)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := providerserver.NewProtocol6(New("test")())()
			schemas, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.ConfigureProvider(context.Background(), &tfprotov6.ConfigureProviderRequest{
				TerraformVersion: "1.16.4", Config: dynamicConfig(t, schemas.Provider.ValueType(), tc.values),
			})
			if err != nil {
				t.Fatal(err)
			}
			if hasError(response.Diagnostics) != tc.wantError {
				t.Fatalf("unexpected diagnostics: %#v", response.Diagnostics)
			}
			for _, diagnostic := range response.Diagnostics {
				if strings.Contains(diagnostic.Detail, "fixture-secret") || strings.Contains(diagnostic.Detail, "explicit-secret") {
					t.Fatal("password leaked in diagnostic")
				}
			}
		})
	}
}

func TestPrinterValidationUnknownAndNull(t *testing.T) {
	for _, tc := range []struct {
		name      string
		values    map[string]any
		wantError bool
	}{
		{"unknown required values", map[string]any{"name": tftypes.UnknownValue, "device_uri": tftypes.UnknownValue}, false},
		{"null optional metadata", map[string]any{"name": "office", "device_uri": "ipp://printer.test/ipp/print"}, false},
		{"invalid queue name", map[string]any{"name": "../office", "device_uri": "ipp://printer.test/ipp/print"}, true},
		{"credential bearing URI", map[string]any{"name": "office", "device_uri": "ipp://user:secret@printer.test/ipp/print"}, true},
		{"oversized metadata", map[string]any{"name": "office", "device_uri": "ipp://printer.test/ipp/print", "location": strings.Repeat("a", 128)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := providerserver.NewProtocol6(New("test")())()
			schemas, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "cups_printer", Config: dynamicConfig(t, schemas.ResourceSchemas["cups_printer"].ValueType(), tc.values),
			})
			if err != nil {
				t.Fatal(err)
			}
			if hasError(response.Diagnostics) != tc.wantError {
				t.Fatalf("unexpected diagnostics: %#v", response.Diagnostics)
			}
			for _, diagnostic := range response.Diagnostics {
				if strings.Contains(diagnostic.Detail, "user:secret") {
					t.Fatal("device URI credentials leaked")
				}
			}
		})
	}
}
