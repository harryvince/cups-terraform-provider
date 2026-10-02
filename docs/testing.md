# Disposable CUPS test environment

The repository includes Docker Compose services for a CUPS server, a simulated IPP Everywhere printer, and a TCP gateway for host access. Docker Engine (or Docker Desktop) and Docker Compose with `up --wait` support are the only host prerequisites. No host CUPS packages or printer hardware are required.

## Start and check

From the repository root:

```sh
docker compose up --build --wait --wait-timeout 120
docker compose exec -T cups python3 /opt/testenv/smoke.py
```

The smoke check sends IPP administrative requests to the Compose server. It checks authentication, creates a uniquely named driverless queue using `ppd-name=everywhere`, verifies a generated PPD, reads and updates its configuration, deletes it, and verifies the not-found response. It does not submit print jobs. This verifies the test environment; it is not a Terraform provider acceptance test, because the provider has not been implemented yet.

## Connection settings

| Setting | Value |
| --- | --- |
| CUPS endpoint from the host | `http://127.0.0.1:8631` |
| CUPS endpoint from the Compose network | `http://cups:631` |
| Admin username | `cups-admin` |
| Default disposable admin password | `cups-test-password` |
| Simulated printer device URI, as seen by CUPS | `ipp://printer.local:8000/ipp/print` |

Open `http://127.0.0.1:8631` for the CUPS web interface. Administrative actions require the test credentials. `printer.local` is a Docker network alias matching the simulator's advertised hostname, not a dependency on host mDNS. The printer URI is resolved by the CUPS container, so the host does not need to resolve it.

Override the host port or test password through environment variables when starting the stack:

```sh
CUPS_TEST_PORT=18631 CUPS_TEST_ADMIN_PASSWORD=another-test-password \
  docker compose up --build --wait --wait-timeout 120
```

These are Compose settings, not a provider configuration contract. Use disposable credentials only. The fixture intentionally permits HTTP Basic authentication without TLS to simplify local tests; it does not verify production TLS behavior.

## Inspect and reset

```sh
docker compose ps
docker compose logs cups printer
docker compose exec -T cups lpstat -h localhost:631 -p
docker compose exec -T cups dpkg-query -W cups
```

`lpstat -p` can return a nonzero exit status when there are no queues; that is normal for a fresh environment. The server starts empty. Test queues live in the container's writable filesystem and survive a stop/start or restart of that same container.

Remove the containers and their state when finished:

```sh
docker compose down
```

Start again with `docker compose up --wait --wait-timeout 120` for a clean server, or add `--build` if fixture files changed. There are no persistent volumes to delete. Images and Docker's build cache are retained for quicker startup.

## Isolation and limitations

- The server is published only on host loopback, at port 8631 by default rather than the usual local CUPS port 631.
- CUPS and the simulator use only an internal Docker network without an external network connection. A TCP gateway connects that network to a separate bridge and forwards the loopback host port to CUPS. This allows host access on Docker versions that do not publish ports for internal-only containers. The simulator has no published host port. Services have no host filesystem mounts, host networking, privileged mode, or host device access.
- Printer discovery advertisements are disabled. Queues should point to the simulated printer; the smoke check uses only the fixed Compose service names.
- Running Docker still uses host disk, CPU, memory, and a loopback port. The stack does not install or configure the host's printing service.
- The image uses Debian Bookworm's CUPS packages. Package revisions and the base-image tag can change on rebuild; this is a repeatable development setup, not a byte-for-byte pinned compatibility matrix. Record the installed version when reporting failures.
- This fixture does not cover TLS verification, other authentication mechanisms, USB devices, legacy PPDs, or physical printer behavior.

Once provider acceptance tests exist, require an explicit opt-in and endpoint. Use this fixture's host endpoint rather than falling back to `localhost:631`.

The initial fixture was tested with Docker Engine 29.8.1, Compose v5.5.1, and CUPS `2.4.2-3+deb12u9` on Linux amd64. Other platforms remain unverified.

## Troubleshooting

If the host port is occupied, select another `CUPS_TEST_PORT`. If a service is unhealthy, inspect `docker compose logs` and rebuild after changing its configuration. Initial image downloads and package installation need internet access during the build; the running services use the isolated network.

The fixture uses [CUPS scheduler configuration](https://openprinting.github.io/cups/doc/man-cupsd.conf.html) and the upstream [IPP printer simulator](https://openprinting.github.io/cups/doc/man-ippeveprinter.html).
