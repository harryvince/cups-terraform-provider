// Package cups manages driverless printer queues through CUPS' IPP API.
package cups

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	ipp "github.com/phin1x/go-ipp"
)

var (
	ErrNotFound = errors.New("queue not found")
	ErrExists   = errors.New("queue already exists; import it instead")
	queueName   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,126}$`)
)

type Printer struct {
	Name        string
	DeviceURI   string
	Description string
	Location    string
}

type Client struct {
	endpoint   *url.URL
	username   string
	password   string
	http       *http.Client
	timeout    time.Duration
	createGate chan struct{}
}

func ValidateName(name string) error {
	if !queueName.MatchString(name) {
		return errors.New("queue name must contain 1–127 ASCII letters, digits, underscores or hyphens and start with a letter or digit")
	}
	return nil
}

func ValidateDeviceURI(value string) error {
	u, err := url.Parse(value)
	if len(value) > 1023 || !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\r\n") || err != nil || u == nil || (u.Scheme != "ipp" && u.Scheme != "ipps") || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return errors.New("device_uri must be an ipp:// or ipps:// URI with a hostname and without credentials, query or fragment")
	}
	return nil
}

func ValidateText(value string) error {
	if len(value) > 127 || !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return errors.New("printer metadata must be valid UTF-8, at most 127 bytes, and contain no NUL characters")
	}
	return nil
}

func validatePrinter(p Printer) error {
	for _, err := range []error{ValidateName(p.Name), ValidateDeviceURI(p.DeviceURI), ValidateText(p.Description), ValidateText(p.Location)} {
		if err != nil {
			return err
		}
	}
	return nil
}

func New(endpoint, username, password string, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("endpoint must be an http:// or https:// server URL without credentials, path, query or fragment")
	}
	if username == "" || password == "" || len(username) > 255 || strings.ContainsAny(username, ":\x00\r\n") {
		return nil, errors.New("an admin username (at most 255 bytes, without colon or control characters) and password are required")
	}
	if timeout < time.Second || timeout > 5*time.Minute {
		return nil, errors.New("request timeout must be between 1 and 300 seconds")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	return &Client{
		endpoint: u, username: username, password: password, timeout: timeout,
		createGate: make(chan struct{}, 1),
		http: &http.Client{Transport: transport, Timeout: timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

func (c *Client) printerURI(name string) string {
	u := *c.endpoint
	u.Scheme = "ipp"
	if c.endpoint.Scheme == "https" {
		u.Scheme = "ipps"
	}
	u.Path = "/printers/" + name
	return u.String()
}

func (c *Client) request(ctx context.Context, operation int16, name string, attrs map[string]any) (*ipp.Response, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	r := ipp.NewRequest(operation, 1)
	r.OperationAttributes[ipp.AttributePrinterURI] = c.printerURI(name)
	r.OperationAttributes[ipp.AttributeRequestingUserName] = c.username
	r.PrinterAttributes = attrs
	if operation == ipp.OperationGetPrinterAttributes {
		r.OperationAttributes[ipp.AttributeRequestedAttributes] = []string{
			ipp.AttributeDeviceURI, ipp.AttributePrinterInfo, ipp.AttributePrinterLocation,
		}
	}
	payload, err := r.Encode()
	if err != nil {
		return nil, errors.New("could not encode IPP request")
	}
	u := *c.endpoint
	u.Path = "/admin/"
	if operation == ipp.OperationGetPrinterAttributes {
		u.Path = "/printers/" + name
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, errors.New("could not construct CUPS request")
	}
	req.Header.Set("Content-Type", ipp.ContentTypeIPP)
	req.SetBasicAuth(c.username, c.password)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, c.transportError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CUPS returned HTTP %d (check server access, admin credentials and encryption requirements)", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024+1))
	if err != nil {
		return nil, c.transportError(ctx, err)
	}
	if len(body) > 4*1024*1024 {
		return nil, errors.New("CUPS response exceeded 4 MiB")
	}
	result, err := decodeResponse(body)
	if err != nil || result == nil || result.RequestId != r.RequestId || (result.ProtocolVersionMajor != 1 && result.ProtocolVersionMajor != 2) {
		return nil, errors.New("invalid IPP response from CUPS")
	}
	if result.StatusCode == ipp.StatusErrorNotFound {
		return nil, ErrNotFound
	}
	if result.StatusCode < 0 || result.StatusCode >= 0x0100 {
		// Server status-message values can echo device URIs or credentials.
		return nil, fmt.Errorf("CUPS returned IPP status 0x%04x", uint16(result.StatusCode))
	}
	return result, nil
}

func decodeResponse(body []byte) (result *ipp.Response, err error) {
	// go-ipp v1.7.0 uses signed length fields and can panic on malformed input.
	// Confine recovery to this third-party decoder, turning it into a diagnostic.
	defer func() {
		if recover() != nil {
			result = nil
			err = errors.New("malformed IPP response")
		}
	}()
	return ipp.NewResponseDecoder(bytes.NewReader(body)).Decode(nil)
}

func (c *Client) transportError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	message := strings.ReplaceAll(err.Error(), c.password, "[redacted]")
	return fmt.Errorf("CUPS transport failed: %s", message)
}

func (c *Client) Read(ctx context.Context, name string) (Printer, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	result, err := c.request(ctx, ipp.OperationGetPrinterAttributes, name, nil)
	if err != nil {
		return Printer{}, err
	}
	if len(result.PrinterAttributes) != 1 {
		return Printer{}, errors.New("CUPS returned an unexpected number of printer attribute groups")
	}
	attrs := result.PrinterAttributes[0]
	get := func(key string, required bool) (string, error) {
		values := attrs[key]
		if len(values) == 0 && !required {
			return "", nil
		}
		if len(values) != 1 {
			return "", fmt.Errorf("CUPS did not return a single %s attribute; administrative access is required", key)
		}
		value, ok := values[0].Value.(string)
		if !ok {
			return "", fmt.Errorf("CUPS returned an invalid %s attribute", key)
		}
		return value, nil
	}
	p := Printer{Name: name}
	if p.DeviceURI, err = get(ipp.AttributeDeviceURI, true); err != nil {
		return Printer{}, err
	}
	if err = ValidateDeviceURI(p.DeviceURI); err != nil {
		return Printer{}, errors.New("queue has an unsupported device URI; this POC supports only credential-free IPP/IPPS devices")
	}
	if p.Description, err = get(ipp.AttributePrinterInfo, false); err != nil {
		return Printer{}, err
	}
	p.Location, err = get(ipp.AttributePrinterLocation, false)
	return p, err
}

func printerAttributes(p Printer) map[string]any {
	return map[string]any{
		ipp.AttributeDeviceURI: p.DeviceURI, ipp.AttributePrinterInfo: p.Description,
		ipp.AttributePrinterLocation: p.Location,
	}
}

// Create returns a nonempty Printer after an acknowledged mutation even if its
// subsequent verification fails, allowing Terraform to retain partial state.
func (c *Client) Create(ctx context.Context, p Printer) (Printer, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if err := validatePrinter(p); err != nil {
		return Printer{}, err
	}
	// Serialize existence checks within this client. CUPS has no atomic create-only
	// operation, so another process can still race this check.
	select {
	case c.createGate <- struct{}{}:
		defer func() { <-c.createGate }()
	case <-ctx.Done():
		return Printer{}, ctx.Err()
	}
	_, err := c.Read(ctx, p.Name)
	if err == nil {
		return Printer{}, ErrExists
	}
	if !errors.Is(err, ErrNotFound) {
		return Printer{}, err
	}
	attrs := printerAttributes(p)
	attrs[ipp.AttributePPDName] = "everywhere"
	if _, err := c.request(ctx, ipp.OperationCupsAddModifyPrinter, p.Name, attrs); err != nil {
		return Printer{}, fmt.Errorf("create request failed: %w; if the request reached CUPS, check the queue and import it before retrying", err)
	}
	if err := c.waitForPPD(ctx, p.Name); err != nil {
		return p, err
	}
	// Driverless PPD generation can fill an empty description with the device's
	// model name. Reapply managed metadata after generation to honor empty values.
	if _, err := c.request(ctx, ipp.OperationCupsAddModifyPrinter, p.Name, map[string]any{
		ipp.AttributePrinterInfo: p.Description, ipp.AttributePrinterLocation: p.Location,
	}); err != nil {
		return p, err
	}
	actual, err := c.Read(ctx, p.Name)
	if err != nil {
		return p, err
	}
	return actual, nil
}

func (c *Client) waitForPPD(ctx context.Context, name string) error {
	u := *c.endpoint
	u.Path = "/printers/" + name + ".ppd"
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return errors.New("could not construct PPD verification request")
		}
		req.SetBasicAuth(c.username, c.password)
		resp, err := c.http.Do(req)
		if err != nil {
			return c.transportError(ctx, err)
		}
		prefix, readErr := io.ReadAll(io.LimitReader(resp.Body, 32))
		resp.Body.Close()
		if readErr != nil {
			return c.transportError(ctx, readErr)
		}
		if resp.StatusCode == http.StatusOK && bytes.HasPrefix(prefix, []byte("*PPD-Adobe:")) {
			return nil
		}
		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("could not verify generated driverless PPD (HTTP %d)", resp.StatusCode)
		}
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("waiting for driverless PPD failed: %w; ensure CUPS can reach the IPP device", ctx.Err())
		case <-timer.C:
		}
	}
}

func (c *Client) Update(ctx context.Context, p Printer) (Printer, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if err := validatePrinter(p); err != nil {
		return Printer{}, err
	}
	previous, err := c.Read(ctx, p.Name)
	if err != nil {
		return Printer{}, err
	}
	if previous.DeviceURI != p.DeviceURI {
		return Printer{}, errors.New("device URI changes require queue replacement")
	}
	attrs := printerAttributes(p)
	if _, err := c.request(ctx, ipp.OperationCupsAddModifyPrinter, p.Name, attrs); err != nil {
		return Printer{}, err
	}
	return c.Read(ctx, p.Name)
}

func (c *Client) Delete(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	_, err := c.request(ctx, ipp.OperationCupsDeletePrinter, name, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}
