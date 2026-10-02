package cups

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ipp "github.com/phin1x/go-ipp"
)

func testClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	c, err := New(endpoint, "admin", "secret-test-password", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func respond(t *testing.T, w http.ResponseWriter, code int16, attrs ipp.Attributes) {
	t.Helper()
	r := ipp.NewResponse(code, 1)
	if attrs != nil {
		r.PrinterAttributes = []ipp.Attributes{attrs}
	}
	data, err := r.Encode()
	if err != nil {
		t.Error(err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", ipp.ContentTypeIPP)
	w.Write(data)
}

func printerResponse() ipp.Attributes {
	return ipp.Attributes{
		ipp.AttributeDeviceURI:       {{Tag: ipp.TagUri, Name: ipp.AttributeDeviceURI, Value: "ipp://printer.test/ipp/print"}},
		ipp.AttributePrinterInfo:     {{Tag: ipp.TagText, Name: ipp.AttributePrinterInfo, Value: "Office"}},
		ipp.AttributePrinterLocation: {{Tag: ipp.TagText, Name: ipp.AttributePrinterLocation, Value: "Floor 1"}},
	}
}

func decodeRequest(r *http.Request) (*ipp.Request, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	// The upstream decoder expects a reader that doesn't return data and EOF
	// together; buffer the HTTP body before decoding, as the real client does.
	return ipp.NewRequestDecoder(bytes.NewReader(body)).Decode(nil)
}

func TestReadMappingAndRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "admin" || password != "secret-test-password" {
			t.Error("missing HTTP authentication")
		}
		if r.URL.Path != "/printers/office" || r.Header.Get("Content-Type") != ipp.ContentTypeIPP {
			t.Error("wrong path/content type")
		}
		request, err := decodeRequest(r)
		if err != nil {
			t.Error(err)
			return
		}
		if request.Operation != ipp.OperationGetPrinterAttributes || request.OperationAttributes[ipp.AttributeRequestingUserName] != "admin" {
			t.Error("wrong IPP operation/user")
		}
		respond(t, w, ipp.StatusOkIgnoredOrSubstituted, printerResponse())
	}))
	defer server.Close()
	p, err := testClient(t, server.URL).Read(context.Background(), "office")
	if err != nil {
		t.Fatal(err)
	}
	if p != (Printer{Name: "office", DeviceURI: "ipp://printer.test/ipp/print", Description: "Office", Location: "Floor 1"}) {
		t.Fatalf("unexpected printer: %#v", p)
	}
}

func TestReadErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		name       string
		httpStatus int
		ippStatus  int16
		notFound   bool
	}{
		{"missing queue", 200, ipp.StatusErrorNotFound, true},
		{"IPP permission failure", 200, ipp.StatusErrorNotAuthorized, false},
		{"HTTP authentication failure", 401, 0, false},
		{"HTTP missing route", 404, 0, false},
		{"HTTP server failure", 503, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.httpStatus != 200 {
					w.WriteHeader(tc.httpStatus)
					w.Write([]byte("secret-test-password"))
					return
				}
				respond(t, w, tc.ippStatus, nil)
			}))
			defer server.Close()
			_, err := testClient(t, server.URL).Read(context.Background(), "office")
			if err == nil || errors.Is(err, ErrNotFound) != tc.notFound {
				t.Fatalf("wrong error classification: %v", err)
			}
			if strings.Contains(err.Error(), "secret-test-password") {
				t.Fatal("credential leaked")
			}
		})
	}
}

func TestReadRejectsInvalidResponses(t *testing.T) {
	for _, payload := range []string{"", "not IPP", "\x02\x00\x00\x00\x00\x00\x00\x02\x03", "\x02\x00\x00\x00\x00\x00\x00\x01\x01\x47\xff\xff"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(payload)) }))
		_, err := testClient(t, server.URL).Read(context.Background(), "office")
		server.Close()
		if err == nil || errors.Is(err, ErrNotFound) {
			t.Fatalf("invalid response treated as valid/missing: %v", err)
		}
	}
}

func TestCreatePreservesExistingQueue(t *testing.T) {
	var writes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeRequest(r)
		if err != nil {
			t.Error(err)
			return
		}
		if req.Operation != ipp.OperationGetPrinterAttributes {
			writes.Add(1)
		}
		respond(t, w, ipp.StatusOk, printerResponse())
	}))
	defer server.Close()
	_, err := testClient(t, server.URL).Create(context.Background(), Printer{Name: "office", DeviceURI: "ipp://printer.test/ipp/print"})
	if !errors.Is(err, ErrExists) || writes.Load() != 0 {
		t.Fatalf("existing queue was not protected: %v, writes=%d", err, writes.Load())
	}
}

func TestDeleteAlreadyMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { respond(t, w, ipp.StatusErrorNotFound, nil) }))
	defer server.Close()
	if err := testClient(t, server.URL).Delete(context.Background(), "office"); err != nil {
		t.Fatal(err)
	}
}

func TestCancellationAndTLSVerification(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := testClient(t, server.URL).Read(ctx, "office")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = testClient(t, server.URL).Read(ctx, "office")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline lost: %v", err)
	}
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { respond(t, w, ipp.StatusOk, printerResponse()) }))
	defer tlsServer.Close()
	_, err = testClient(t, tlsServer.URL).Read(context.Background(), "office")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatal("untrusted TLS certificate was accepted or treated as missing")
	}
	trusted := testClient(t, tlsServer.URL)
	trusted.http.Transport = tlsServer.Client().Transport
	if _, err := trusted.Read(context.Background(), "office"); err != nil {
		t.Fatalf("trusted TLS failed: %v", err)
	}
}

func TestCreateReturnsPartialStateOnPPDTimeout(t *testing.T) {
	var mutations atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		req, err := decodeRequest(r)
		if err != nil {
			t.Error(err)
			return
		}
		if req.Operation == ipp.OperationCupsAddModifyPrinter {
			mutations.Add(1)
			respond(t, w, ipp.StatusOk, nil)
		} else {
			respond(t, w, ipp.StatusErrorNotFound, nil)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	wanted := Printer{Name: "office", DeviceURI: "ipp://printer.test/ipp/print"}
	actual, err := testClient(t, server.URL).Create(ctx, wanted)
	if !errors.Is(err, context.DeadlineExceeded) || actual != wanted || mutations.Load() != 1 {
		t.Fatalf("create did not preserve acknowledged partial state without replaying mutation: %#v, %v, writes=%d", actual, err, mutations.Load())
	}
}

func TestRedirectDoesNotForwardCredentials(t *testing.T) {
	var requests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests.Add(1) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer server.Close()
	_, err := testClient(t, server.URL).Read(context.Background(), "office")
	if err == nil || requests.Load() != 0 {
		t.Fatal("redirect was followed")
	}
}

func TestValidateInputs(t *testing.T) {
	for _, value := range []string{"", "../office", "office/printer", "office printer", strings.Repeat("a", 128)} {
		if ValidateName(value) == nil {
			t.Errorf("accepted queue name %q", value)
		}
	}
	for _, value := range []string{"ipp://printer.test/ipp/print", "ipps://printer.test:631/ipp/print"} {
		if err := ValidateDeviceURI(value); err != nil {
			t.Error(err)
		}
	}
	for _, value := range []string{"socket://printer.test", "ipp://user:password@printer.test/ipp/print", "ipp:///ipp/print", "ipp://printer.test/ipp/print?password=secret"} {
		if ValidateDeviceURI(value) == nil {
			t.Errorf("accepted unsupported URI %q", value)
		}
	}
	for _, value := range []string{"http://user:password@cups.test", "http://cups.test/admin", "http://cups.test?password=secret", "ipp://cups.test", ""} {
		if _, err := New(value, "admin", "password", time.Second); err == nil {
			t.Errorf("accepted endpoint %q", value)
		}
	}
}
