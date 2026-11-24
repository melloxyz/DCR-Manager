"""Read-only TOML MCP audit. Emits counts and fixed classifications, never raw config."""
import json
import re
import sys
import tomllib
from pathlib import Path

def is_dcr(value):
    if not isinstance(value, str):
        return False
    return bool(re.fullmatch(r"@wonderwhy-er/desktop-commander(?:@[^@\s]+)?", value)
                or re.search(r"[/\\]+@wonderwhy-er[/\\]+desktop-commander[/\\]+", value, re.I))

def summarize(data):
    servers = data.get("mcp_servers", {})
    if not isinstance(servers, dict):
        raise ValueError("unsupported")
    result = []
    for server in servers.values():
        if not isinstance(server, dict):
            continue
        command = server.get("command", "")
        args = server.get("args", [])
        if not isinstance(command, str) or not isinstance(args, list):
            continue
        if not (is_dcr(command) or any(is_dcr(a) for a in args)):
            continue
        executable = re.split(r"[/\\]", command)[-1].lower()
        mode = "npx" if executable in ("npx", "npx.cmd", "npx.exe") else "direct" if executable in ("node", "node.exe") else "other"
        result.append({"mode": mode, "floatingVersion": any(isinstance(a, str) and re.fullmatch(r"@wonderwhy-er/desktop-commander(?:@latest)?", a) for a in args)})
    return result

def main():
    try:
        path = Path(sys.argv[1])
        if path.stat().st_size > 5 * 1024 * 1024:
            raise ValueError("too large")
        with path.open("rb") as file:
            result = summarize(tomllib.load(file))
        print(json.dumps(result))
        return 0
    except Exception:
        print("[]")
        return 2

if __name__ == "__main__":
    sys.exit(main())
