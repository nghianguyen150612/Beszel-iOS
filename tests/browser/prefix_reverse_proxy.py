#!/usr/bin/env python3
"""Loopback-only HTTP reverse proxy for Hub subpath browser tests."""

from __future__ import annotations

import argparse
from http.client import HTTPConnection, HTTPSConnection
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlsplit


HOP_HEADERS = {"connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailers", "transfer-encoding", "upgrade"}


def loopback(host: str | None) -> bool:
    return host in {"127.0.0.1", "localhost", "::1"}


def serve(upstream: str, listen: str, prefix: str) -> None:
    target = urlsplit(upstream)
    if target.scheme not in {"http", "https"} or not loopback(target.hostname) or target.path not in {"", "/"}:
        raise SystemExit("upstream must be a loopback HTTP(S) origin")
    listen_host, listen_port = listen.rsplit(":", 1)
    if not loopback(listen_host):
        raise SystemExit("proxy listener must use a loopback address")
    prefix = "/" + prefix.strip("/")

    class ProxyHandler(BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def do_GET(self):
            self.forward()

        def do_HEAD(self):
            self.forward()

        def do_POST(self):
            self.forward()

        def do_PATCH(self):
            self.forward()

        def do_PUT(self):
            self.forward()

        def do_DELETE(self):
            self.forward()

        def forward(self) -> None:
            if self.path == prefix:
                path = "/"
            elif self.path.startswith(prefix + "/"):
                path = self.path[len(prefix) :]
            else:
                self.send_error(404)
                return
            if self.headers.get("Upgrade", "").lower() == "websocket":
                self.send_error(501, "WebSocket forwarding is outside this HTTP asset test")
                return

            connection_type = HTTPSConnection if target.scheme == "https" else HTTPConnection
            connection = connection_type(target.hostname, target.port or (443 if target.scheme == "https" else 80), timeout=20)
            body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
            headers = {
                key: value
                for key, value in self.headers.items()
                if key.lower() not in HOP_HEADERS and key.lower() != "host"
            }
            headers["Host"] = target.netloc
            try:
                connection.request(self.command, path, body=body if body else None, headers=headers)
                response = connection.getresponse()
                self.send_response(response.status, response.reason)
                streaming = "text/event-stream" in response.getheader("Content-Type", "").lower()
                for key, value in response.getheaders():
                    if key.lower() not in HOP_HEADERS and key.lower() != "content-length":
                        self.send_header(key, value)
                if not streaming:
                    payload = response.read()
                    self.send_header("Content-Length", str(len(payload)))
                else:
                    payload = b""
                    self.send_header("Transfer-Encoding", "chunked")
                self.end_headers()
                if streaming and self.command != "HEAD":
                    while True:
                        chunk = response.read1(4096)
                        if not chunk:
                            break
                        self.wfile.write(f"{len(chunk):X}\r\n".encode("ascii"))
                        self.wfile.write(chunk)
                        self.wfile.write(b"\r\n")
                        self.wfile.flush()
                    self.wfile.write(b"0\r\n\r\n")
                    self.wfile.flush()
                elif self.command != "HEAD":
                    self.wfile.write(payload)
            except (BrokenPipeError, ConnectionResetError, OSError):
                return
            finally:
                connection.close()

        def log_message(self, _format: str, *_args) -> None:
            return

    server = ThreadingHTTPServer((listen_host, int(listen_port)), ProxyHandler)
    print(f"loopback proxy http://{listen}{prefix}/ -> {upstream}/", flush=True)
    server.serve_forever()


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--upstream", default="http://127.0.0.1:8097")
    parser.add_argument("--listen", default="127.0.0.1:8098")
    parser.add_argument("--prefix", default="/beszel")
    args = parser.parse_args()
    serve(args.upstream, args.listen, args.prefix)


if __name__ == "__main__":
    main()
