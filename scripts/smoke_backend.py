"""Exercise the built Linux backend process without creating database files."""

import http.client
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
                connection.request("GET", "/")
                response = connection.getresponse()
                response.read()
                assert response.status == 404, response.status
                break
            except OSError:
                if time.monotonic() >= deadline:
                    raise AssertionError("server did not start within 10 seconds")
                time.sleep(0.05)
            finally:
                connection.close()
        process.send_signal(signal.SIGTERM)
        output, _ = process.communicate(timeout=7)
        assert process.returncode == 0, (process.returncode, output)
        assert f"127.0.0.1:{port}" in output, output
        with socket.socket() as probe:
            probe.bind(("127.0.0.1", port))
        print("PASS: invalid config; occupied port; loopback startup; default 404; SIGTERM exit 0; port released")
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
