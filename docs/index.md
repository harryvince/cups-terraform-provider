# CUPS provider settings

The provider manages queue configuration on an existing CUPS server. It uses HTTP Basic authentication and IPP administrative operations. No package installation, service restart, print job submission, or discovery happens during configuration or planning.

Use the selected source address `registry.terraform.io/harryvince/cups` with a [development override](development.md). This provider is not published.

| Setting | Type | Behavior |
| --- | --- | --- |
| `endpoint` | String | HTTP(S) server root URL, or `CUPS_ENDPOINT` when omitted/null. Required after fallback; no default server. |
| `username` | String | Admin username, or `CUPS_USERNAME` when omitted/null. Required after fallback. |
| `password` | Sensitive string | Admin password, or `CUPS_PASSWORD` when omitted/null. Required after fallback. |
| `request_timeout` | Integer | 1–300 seconds per lifecycle operation, including PPD generation; default 30. |

Explicit configuration takes precedence over environment variables. An explicit empty string is an error rather than a request to use the environment. Connection settings must be known before applying resources.

An endpoint must contain a hostname and must not contain credentials, a non-root path, query, or fragment. For example, use `http://127.0.0.1:8631` for the fixture, or an explicit `https://cups.example.test:631` server for TLS. HTTPS uses normal system certificate verification; there is no verification-bypass option. HTTP is explicitly supported for the local fixture. Unix sockets, TLS upgrade negotiation, client certificates, and Kerberos are not implemented.

Prefer environment variables for credentials. Marking `password` sensitive hides normal Terraform display but does not prevent a configured value from being stored in saved plans. Resource state contains queue identity, device URI, description, and location; it has no admin credential fields. Diagnostics omit server-provided status text because it can echo credentials.

One provider instance targets one server. Use aliases for separate servers. Changing an endpoint redirects reads and mutations for existing state to another server; perform a deliberate state/import migration rather than repointing a provider configuration with managed queues.

The POC's public schema is provisional. Compatibility has only been verified in the documented [test environment](testing.md).
