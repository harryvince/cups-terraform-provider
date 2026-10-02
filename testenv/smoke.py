"""Exercise only the Compose CUPS server; never submit a print job."""

import base64
import http.client
import os
import struct
import time
import uuid


QUEUE = "smoke-" + uuid.uuid4().hex[:12]
URI = f"ipp://cups:631/printers/{QUEUE}"
PASSWORD = os.environ["CUPS_TEST_ADMIN_PASSWORD"]
AUTH = "Basic " + base64.b64encode(
    f"cups-admin:{PASSWORD}".encode()
).decode()


def attribute(tag, name, value):
    name = name.encode()
    value = value.encode()
    return bytes([tag]) + struct.pack(">H", len(name)) + name + struct.pack(">H", len(value)) + value


def request(operation, attributes=(), *, authenticated=True, path="/admin/"):
    body = struct.pack(">BBHI", 2, 0, operation, 1) + b"\x01"
    body += attribute(0x47, "attributes-charset", "utf-8")
    body += attribute(0x48, "attributes-natural-language", "en")
    body += attribute(0x45, "printer-uri", URI)
    body += attribute(0x42, "requesting-user-name", "cups-admin")
    if attributes:
        body += b"\x04"
        for tag, name, value in attributes:
            body += attribute(tag, name, value)
    body += b"\x03"
    headers = {"Content-Type": "application/ipp"}
    if authenticated:
        headers["Authorization"] = AUTH
    connection = http.client.HTTPConnection("cups", 631, timeout=10)
    try:
        connection.request("POST", path, body, headers)
        response = connection.getresponse()
        data = response.read()
        if response.status != 200:
            return response.status, None, data
        if len(data) < 8 or data[:2] != b"\x02\x00":
            raise RuntimeError("Invalid IPP response")
        return response.status, struct.unpack(">H", data[2:4])[0], data
    finally:
        connection.close()


def success(result):
    http_status, ipp_status, _ = result
    assert http_status == 200 and ipp_status is not None and ipp_status < 0x0100, (
        f"Operation failed: HTTP {http_status}, IPP {ipp_status}"
    )


def main():
    # A requesting-user-name alone must not grant administrative access.
    status, _, _ = request(0x4003, authenticated=False)
    assert status == 401, f"Expected unauthenticated create to return 401, got {status}"
    print("PASS: unauthenticated administrative request rejected", flush=True)

    created = False
    try:
        success(request(0x4003, [
            (0x45, "device-uri", "ipp://printer.local:8000/ipp/print"),
            (0x42, "ppd-name", "everywhere"),
            (0x41, "printer-info", "Smoke test printer"),
            (0x41, "printer-location", "Test room"),
        ]))
        created = True
        # CUPS can acknowledge creation before driverless PPD generation finishes.
        deadline = time.monotonic() + 15
        while True:
            connection = http.client.HTTPConnection("cups", 631, timeout=5)
            try:
                connection.request("GET", f"/printers/{QUEUE}.ppd", headers={"Authorization": AUTH})
                response = connection.getresponse()
                ppd = response.read()
                if response.status == 200 and ppd.startswith(b"*PPD-Adobe:"):
                    break
                if response.status != 404 or time.monotonic() >= deadline:
                    raise RuntimeError("Expected a generated driverless PPD, not a raw queue")
            finally:
                connection.close()
            time.sleep(0.2)
        print("PASS: authenticated driverless queue creation", flush=True)
        result = request(0x000B, path=f"/printers/{QUEUE}")
        success(result)
        assert b"Smoke test printer" in result[2] and b"Test room" in result[2]
        assert b"ipp://printer.local:8000/ipp/print" in result[2]
        print("PASS: queue read", flush=True)
        success(request(0x4003, [(0x41, "printer-location", "Updated room")]))
        result = request(0x000B, path=f"/printers/{QUEUE}")
        success(result)
        assert b"Updated room" in result[2] and b"Test room" not in result[2]
        print("PASS: queue update and read-back", flush=True)
        success(request(0x4004))
        created = False
        result = request(0x000B, path=f"/printers/{QUEUE}")
        assert result[0] == 200 and result[1] == 0x0406, "Expected IPP not-found after deletion"
        print("PASS: queue deletion and not-found response", flush=True)
    finally:
        if created:
            success(request(0x4004))


if __name__ == "__main__":
    main()
