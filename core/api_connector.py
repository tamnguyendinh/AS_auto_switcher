"""
API Connector module for Antigravity Language Server.
Handles process discovery, port detection, and CSRF token extraction.
"""

import os
import re
import json
import logging
import subprocess
import urllib.request
import urllib.error
from dataclasses import dataclass
from typing import Optional, List, Dict, Any, Callable

logger = logging.getLogger(__name__)


@dataclass
class LanguageServerInfo:
    """Information required to communicate with Antigravity Language Server."""
    pid: int
    csrf_token: str
    port: int

    @property
    def base_url(self) -> str:
        """Returns loopback base URL for HTTP communication."""
        return f"http://127.0.0.1:{self.port}"


def extract_csrf_token(cmdline: Optional[str]) -> Optional[str]:
    """
    Extract CSRF token UUID from language server command-line string.
    Supports formats:
      --csrf_token <UUID>
      --csrf_token=<UUID>
      --csrf_token "<UUID>"
      --csrf_token='<UUID>'
    """
    if not cmdline or not isinstance(cmdline, str):
        return None

    pattern = r"--csrf_token(?:=|\s+)(?:[\"']?)([a-zA-Z0-9_-]+)(?:[\"']?)"
    match = re.search(pattern, cmdline)
    if match:
        token = match.group(1).strip("\"'")
        return token if token else None
    return None


def probe_http_port(port: int, host: str = "127.0.0.1", timeout: float = 0.2) -> bool:
    """
    Probe loopback port to check if it responds with HTTP status 200.
    """
    url = f"http://{host}:{port}/"
    try:
        req = urllib.request.Request(url, method="GET")
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status == 200
    except urllib.error.HTTPError as e:
        return e.code == 200
    except Exception:
        return False


def discover_listening_ports(
    pid: int,
    netstat_cmd: Optional[List[str]] = None,
    ps_cmd: Optional[str] = None,
) -> List[int]:
    """
    Discover all loopback TCP listening ports owned by the given PID.
    Uses netstat first for fast resolution (~15ms), falling back to PowerShell Get-NetTCPConnection.
    """
    if not pid or pid <= 0:
        return []

    ports: List[int] = []

    # Method 1: Fast netstat query
    cmd = netstat_cmd or ["netstat", "-ano", "-p", "tcp"]
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=3.0)
        for line in proc.stdout.splitlines():
            parts = line.strip().split()
            if len(parts) >= 5:
                proto, local_addr, _, state, owner_pid = parts[0], parts[1], parts[2], parts[3], parts[4]
                if proto.upper() == "TCP" and state.upper() == "LISTENING" and owner_pid == str(pid):
                    idx = local_addr.rfind(":")
                    if idx != -1:
                        try:
                            port = int(local_addr[idx + 1:])
                            host = local_addr[:idx].strip("[]")
                            if host in ("127.0.0.1", "0.0.0.0", "::1", "::", "*", "localhost") or not host:
                                if port not in ports:
                                    ports.append(port)
                        except ValueError:
                            pass
    except Exception as exc:
        logger.debug("netstat inspection failed: %s", exc)

    # Method 2: Fallback to PowerShell Get-NetTCPConnection
    if not ports:
        ps_script = ps_cmd or f"Get-NetTCPConnection -OwningProcess {pid} -State Listen | Select-Object -ExpandProperty LocalPort"
        full_ps_cmd = [
            "powershell",
            "-NoProfile",
            "-NonInteractive",
            "-ExecutionPolicy",
            "Bypass",
            "-Command",
            ps_script,
        ]
        try:
            proc = subprocess.run(full_ps_cmd, capture_output=True, text=True, timeout=5.0)
            for item in proc.stdout.strip().split():
                try:
                    port = int(item)
                    if port not in ports:
                        ports.append(port)
                except ValueError:
                    pass
        except Exception as exc:
            logger.debug("PowerShell TCP connection query failed: %s", exc)

    return sorted(ports)


def discover_listening_port(
    pid: int,
    ports_provider: Optional[Callable[[int], List[int]]] = None,
) -> Optional[int]:
    """
    Discover the loopback TCP listening port owned by the PID.
    If multiple listening ports are found (e.g. HTTP and HTTPS/gRPC),
    probes and selects the HTTP port that responds with 200 OK.
    """
    if not pid or pid <= 0:
        return None

    ports = ports_provider(pid) if ports_provider else discover_listening_ports(pid)
    if not ports:
        return None

    if len(ports) == 1:
        return ports[0]

    # Prioritize port that responds to HTTP 200
    for p in ports:
        if probe_http_port(p):
            return p

    # Fallback to lowest port if probe fails or in non-network environments
    return ports[0]


def query_running_language_server_processes() -> List[Dict[str, Any]]:
    """
    Query Win32_Process for running language_server.exe instances using CIM.
    Returns list of dicts with 'ProcessId' and 'CommandLine'.
    """
    ps_cmd = (
        "Get-CimInstance Win32_Process -Filter \"Name = 'language_server.exe'\" | "
        "Select-Object ProcessId, CommandLine | ConvertTo-Json"
    )
    full_cmd = [
        "powershell",
        "-NoProfile",
        "-NonInteractive",
        "-ExecutionPolicy",
        "Bypass",
        "-Command",
        ps_cmd,
    ]
    try:
        proc = subprocess.run(full_cmd, capture_output=True, text=True, timeout=5.0)
        raw = proc.stdout.strip()
        if not raw:
            return []
        data = json.loads(raw)
        if isinstance(data, dict):
            return [data]
        elif isinstance(data, list):
            return data
        return []
    except Exception as exc:
        logger.debug("Failed querying Win32_Process: %s", exc)
        return []


def find_language_server_process(
    process_provider: Optional[Callable[[], List[Dict[str, Any]]]] = None,
    port_provider: Optional[Callable[[int], Optional[int]]] = None,
) -> Optional[LanguageServerInfo]:
    """
    Scan running processes for language_server.exe, extract CSRF token and loopback listening port.
    Returns LanguageServerInfo if found, or None if process is absent or invalid.
    """
    query_func = process_provider or query_running_language_server_processes
    port_func = port_provider or discover_listening_port

    try:
        procs = query_func()
    except Exception as exc:
        logger.debug("Error retrieving process candidates: %s", exc)
        return None

    for proc_info in procs:
        pid = proc_info.get("ProcessId")
        cmdline = proc_info.get("CommandLine")

        if not pid or not isinstance(pid, int) or pid <= 0:
            continue
        if not cmdline or not isinstance(cmdline, str):
            continue

        token = extract_csrf_token(cmdline)
        if not token:
            continue

        port = port_func(pid)
        if port and port > 0:
            return LanguageServerInfo(pid=pid, csrf_token=token, port=port)

    return None
