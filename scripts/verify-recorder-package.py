import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import tarfile
import tempfile


def verify(archive, offline_setup):
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    checksum = Path(str(archive) + ".sha256").read_text().split()
    if checksum != [digest, archive.name]:
        raise ValueError("Archive checksum does not match")
    with tempfile.TemporaryDirectory(prefix="recorder-verify-") as temporary:
        root = Path(temporary)
        name = archive.name.removesuffix(".tar.gz")
        with tarfile.open(archive) as source:
            for item in source.getmembers():
                if (item.name != name and not item.name.startswith(name + "/")) or ".." in Path(item.name).parts or not (item.isfile() or item.isdir()):
                    raise ValueError("Archive contains an unexpected path or file type")
            source.extractall(root)
        package = root / name
        manifest = json.loads((package / "manifest.json").read_text())
        expected = {"teststudio-recorder", "scripts/record-browser.mjs", "runners/playwright/package.json", "runners/playwright/package-lock.json", "setup-recorder.mjs", "README.txt"}
        if manifest["schema"] != 1 or {row["path"] for row in manifest["files"]} != expected:
            raise ValueError("Package does not contain the required recorder files")
        if {str(item.relative_to(package)) for item in package.rglob("*") if item.is_file()} != expected | {"manifest.json"}:
            raise ValueError("Package contains unexpected files")
        workspace = Path(__file__).resolve().parent.parent
        for row in manifest["files"]:
            item = package / row["path"]
            data = item.read_bytes()
            if len(data) != row["bytes"] or hashlib.sha256(data).hexdigest() != row["sha256"]:
                raise ValueError("Packaged file checksum does not match")
            if row["path"] in {"scripts/record-browser.mjs", "runners/playwright/package.json", "runners/playwright/package-lock.json"} and data != (workspace / row["path"]).read_bytes():
                raise ValueError("Packaged helper or dependencies differ from the trusted workspace")
        executable = package / "teststudio-recorder"
        if executable.stat().st_mode & 0o777 != 0o755:
            raise ValueError("Recorder command is not executable")
        architecture = {"aarch64": "arm64", "x86_64": "amd64"}.get(platform.machine(), platform.machine())
        native = manifest["platform"] == platform.system().lower() and manifest["architecture"] == architecture
        if native:
            help_text = subprocess.check_output([str(executable), "--help"], text=True)
            if not all(flag in help_text for flag in ["-server", "-resume", "-helper", "-playwright-cli"]):
                raise ValueError("Packaged command does not expose the required desktop options")
        if offline_setup:
            if not native:
                raise ValueError("Offline desktop setup requires a native package")
            environment = {**os.environ, "npm_config_offline": "true", "HTTP_PROXY": "http://127.0.0.1:9", "HTTPS_PROXY": "http://127.0.0.1:9", "ALL_PROXY": "http://127.0.0.1:9"}
            subprocess.run(["node", "setup-recorder.mjs"], cwd=package, env=environment, check=True, timeout=60)
            version = subprocess.check_output(["node", "runners/playwright/node_modules/playwright/cli.js", "--version"], cwd=package, text=True).strip()
            if version != "Version " + manifest["playwrightVersion"]:
                raise ValueError("Offline setup installs a different Playwright version")
        result = {"archive": str(archive), "sha256": digest, "files": len(expected), "platform": manifest["platform"], "architecture": manifest["architecture"], "nativeHelp": native, "offlineSetup": offline_setup, "playwrightVersion": manifest["playwrightVersion"]}
    result["temporaryVerificationRemoved"] = not root.exists()
    Path(str(archive) + ".verification.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result))


parser = argparse.ArgumentParser(description="Verify a standalone TestStudio desktop recorder archive without cloud operations")
parser.add_argument("archive", type=Path)
parser.add_argument("--offline-setup", action="store_true", help="Use cached npm dependencies and Chromium to verify setup without downloading dependencies")
arguments = parser.parse_args()
verify(arguments.archive.resolve(), arguments.offline_setup)
