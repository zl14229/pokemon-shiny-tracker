"""Exercise the built Linux backend process without creating database files."""

import http.client
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]
BINARY = ROOT / "bin/shiny-tracker"


def start(port):
    return subprocess.Popen(
        [str(BINARY)], cwd=ROOT,
        env=dict(os.environ, SHINY_HTTP_PORT=str(port)),
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
    )


def expect_failure(port):
    process = start(port)
    try:
        output, _ = process.communicate(timeout=5)
        if process.returncode != 1:
            raise AssertionError(f"expected exit 1, got {process.returncode}: {output}")
    finally:
        if process.poll() is None:
            process.kill()
            process.communicate()


def main():
    if not BINARY.is_file():
        raise AssertionError("build bin/shiny-tracker before running the smoke check")
    expect_failure("invalid-port")
    with socket.socket() as occupied:
        occupied.bind(("127.0.0.1", 0))
        occupied.listen()
        port = occupied.getsockname()[1]
        expect_failure(port)
    # Release this disposable socket before the subprocess binds the same port.
    process = start(port)
    try:
        deadline = time.monotonic() + 10
        while True:
            if process.poll() is not None:
                raise AssertionError(f"server exited early: {process.communicate()[0]}")
            connection = http.client.HTTPConnection("127.0.0.1", port, timeout=0.5)
            try:
                connection.request("GET", "/api/v1/health")
                response = connection.getresponse()
                body = json.loads(response.read())
                assert response.status == 200, response.status
                assert body == {"data": {"status": "ok"}}, body
                break
            except OSError:
                if time.monotonic() >= deadline:
                    raise AssertionError("server did not start within 10 seconds")
                time.sleep(0.05)
            finally:
                connection.close()
        for method, path, status, code in (
            ("GET", "/api/v1/missing", 404, "not_found"),
            ("POST", "/api/v1/health", 405, "method_not_allowed"),
        ):
            connection = http.client.HTTPConnection("127.0.0.1", port, timeout=1)
            try:
                connection.request(method, path)
                response = connection.getresponse()
                body = json.loads(response.read())
                assert response.status == status, response.status
                assert body["error"]["code"] == code, body
            finally:
                connection.close()
        process.send_signal(signal.SIGTERM)
        output, _ = process.communicate(timeout=7)
        assert process.returncode == 0, (process.returncode, output)
        assert f"127.0.0.1:{port}" in output, output
        with socket.socket() as probe:
            # Match Go's TCP listener behavior: TIME_WAIT is not a live listener.
            probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            probe.bind(("127.0.0.1", port))
        print("PASS: invalid config; occupied port; loopback health 200; JSON 404/405; SIGTERM exit 0; port released")
    finally:
        if process.poll() is None:
            process.kill()
            process.communicate()


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, OSError, subprocess.TimeoutExpired) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        sys.exit(1)
