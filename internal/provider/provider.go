package provider

import (
	"context"
	"os"
	"time"

	"github.com/harryvince/terraform-provider-cups/internal/cups"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type cupsProvider struct{ version string }

type providerModel struct {
	Endpoint       types.String `tfsdk:"endpoint"`
	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	RequestTimeout types.Int64  `tfsdk:"request_timeout"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &cupsProvider{version: version} }
}

func (p *cupsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cups"
	resp.Version = p.version
}

func (p *cupsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage driverless printer queues on an existing CUPS server.",
		Attributes: map[string]schema.Attribute{
			"endpoint":        schema.StringAttribute{Optional: true, Description: "Explicit HTTP(S) CUPS server URL. Defaults to CUPS_ENDPOINT; no implicit local server."},
			"username":        schema.StringAttribute{Optional: true, Description: "Admin username. Defaults to CUPS_USERNAME."},
			"password":        schema.StringAttribute{Optional: true, Sensitive: true, Description: "Admin password. Defaults to CUPS_PASSWORD. Prefer the environment variable to avoid including it in configuration or plans."},
			"request_timeout": schema.Int64Attribute{Optional: true, Description: "Maximum seconds per lifecycle operation, including PPD verification; 1–300, default 30."},
		},
	}
}

func (p *cupsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for name, unknown := range map[string]bool{
		"endpoint": data.Endpoint.IsUnknown(), "username": data.Username.IsUnknown(),
		"password": data.Password.IsUnknown(), "request_timeout": data.RequestTimeout.IsUnknown(),
	} {
		if unknown {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Unknown provider configuration", "CUPS connection settings must be known before applying printer resources.")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	value := func(v types.String, env string) string {
		if v.IsNull() {
			return os.Getenv(env)
		}
		return v.ValueString()
	}
	endpoint := value(data.Endpoint, "CUPS_ENDPOINT")
	username := value(data.Username, "CUPS_USERNAME")
	password := value(data.Password, "CUPS_PASSWORD")
	for name, v := range map[string]string{"endpoint": endpoint, "username": username, "password": password} {
		if v == "" {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Missing CUPS connection setting", "Set this attribute or its documented environment variable. No local server or credentials are assumed.")
		}
	}
	timeout := int64(30)
	if !data.RequestTimeout.IsNull() {
		timeout = data.RequestTimeout.ValueInt64()
	}
	if timeout < 1 || timeout > 300 {
		resp.Diagnostics.AddAttributeError(path.Root("request_timeout"), "Invalid timeout", "request_timeout must be between 1 and 300 seconds.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := cups.New(endpoint, username, password, time.Duration(timeout)*time.Second)
	if err != nil {
		resp.Diagnostics.AddError("Invalid CUPS configuration", err.Error())
		return
	}
	resp.ResourceData = client
}

func (p *cupsProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return &printerResource{} }}
}

func (p *cupsProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }
