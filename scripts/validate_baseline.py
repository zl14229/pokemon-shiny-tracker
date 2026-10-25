"""Validate repository governance using only Python's standard library and Git."""

import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
ERRORS = []


def require(condition, message):
    if not condition:
        ERRORS.append(message)


def git(*args, input=None):
    return subprocess.run(
        ["git", *args], cwd=ROOT, input=input, text=True,
        encoding="utf-8", capture_output=True, check=False,
    )


def validate():
    manifest = json.loads((ROOT / "docs/baseline-manifest.json").read_text("utf-8"))
    expected_files = {
        "AGENTS.md", "README_BASELINE.md", "docs/PRODUCT_SPEC_V1.1.md",
        "docs/TASK_BASELINE_V1.1.md", "docs/ADR_REGISTER_V1.1.md",
        "docs/DATA_SOURCE_POLICY_V1.1.md",
    }
    require(set(manifest["files"]) == expected_files, "Frozen file set must contain six originals")
    for name, digest in manifest["files"].items():
        path = ROOT / name
        require(path.is_file(), f"Missing frozen document: {name}")
        if path.is_file():
            require(hashlib.sha256(path.read_bytes()).hexdigest() == digest,
                    f"Frozen document changed: {name}; requires reviewed baseline change")

    text = (ROOT / "docs/TASK_BASELINE_V1.1.md").read_text("utf-8")
    definitions = {}
    for line in text.splitlines():
        if re.match(r"\| T\d{3} \|", line):
            task_id, milestone, deliverable, deps, acceptance = [s.strip() for s in line.split("|")[1:-1]]
            require(task_id not in definitions, f"Duplicate baseline task {task_id}")
            definitions[task_id] = dict(
                milestone=milestone, deliverable=deliverable,
                dependencies=[] if deps == "无" else deps.split(","), acceptance=acceptance,
            )
    expected_ids = {f"T{i:03d}" for i in range(29)}
    require(set(definitions) == expected_ids, "Baseline must contain exactly T000–T028")
    registry = json.loads((ROOT / "docs/tasks/status.json").read_text("utf-8"))
    tasks = {task["id"]: task for task in registry["tasks"]}
    require(len(registry["tasks"]) == len(tasks) == 29, "Registry needs 29 unique tasks")
    require(set(tasks) == expected_ids, "Registry task IDs differ from baseline")
    states = {"planned", "in_progress", "in_review", "verified", "blocked", "needs_decision"}
    for task_id, task in tasks.items():
        definition = definitions.get(task_id)
        require(definition is not None, f"Unknown task {task_id}")
        if definition:
            for field, value in definition.items():
                require(task.get(field) == value, f"{task_id}: baseline {field} mismatch")
        require(task.get("status") in states, f"{task_id}: invalid state")
        require(set(task["dependencies"]) <= expected_ids, f"{task_id}: unknown dependency")
        require(task_id not in task["dependencies"], f"{task_id}: self-dependency")
        if task.get("record"):
            require((ROOT / task["record"]).is_file(), f"{task_id}: missing task record")
        if task.get("branch"):
            require(re.fullmatch(rf"feature/{task_id}-[a-z0-9]+(?:-[a-z0-9]+)*", task["branch"]),
                    f"{task_id}: invalid branch name")
        if task.get("status") in {"in_progress", "in_review", "verified"}:
            require(all(tasks.get(d, {}).get("status") == "verified" for d in task["dependencies"]),
                    f"{task_id}: dependencies not verified")
        if task.get("status") == "verified":
            evidence = task.get("evidence", {})
            for field in ["implementation", "checks", "self_review", "documentation"]:
                require(bool(evidence.get(field)), f"{task_id}: missing evidence {field}")
            for field in ["commit", "merge_sha"]:
                require(bool(re.fullmatch(r"[0-9a-f]{40}", evidence.get(field) or "")),
                        f"{task_id}: invalid {field}")
            require(bool(re.fullmatch(r"https://github\.com/zl14229/pokemon-shiny-tracker/pull/\d+",
                                      evidence.get("pr_url") or "")), f"{task_id}: missing canonical PR")

    visiting, visited = set(), set()

    def visit(task_id):
        if task_id in visiting:
            ERRORS.append(f"Dependency cycle at {task_id}")
            return
        if task_id in visited:
            return
        visiting.add(task_id)
        for dep in definitions.get(task_id, {}).get("dependencies", []):
            visit(dep)
        visiting.remove(task_id)
        visited.add(task_id)

    for task_id in definitions:
        visit(task_id)
    adr = (ROOT / "docs/ADR_REGISTER_V1.1.md").read_text("utf-8")
    adr_ids = re.findall(r"^\| (ADR-\d{3}) \|", adr, re.M)
    require(len(adr_ids) == 16 and set(adr_ids) == {f"ADR-{i:03d}" for i in range(1, 17)},
            "ADR register needs exactly ADR-001–016")
    for name in expected_files:
        for task_id in re.findall(r"\bT\d{3}\b", (ROOT / name).read_text("utf-8")):
            require(task_id in expected_ids, f"{name}: unknown task {task_id}")

    required = [".github/pull_request_template.md", ".github/workflows/ci.yml", ".gitignore",
                "docs/GITHUB_WORKFLOW.md", "docs/tasks/TEMPLATE.md", "docs/tasks/T000.md",
                "docs/T000_ACCEPTANCE.md"]
    for name in required:
        require((ROOT / name).is_file(), f"Missing governance file {name}")

    # Check ignore behavior without creating real secrets or personal data.
    probes = [".env", ".env.local", "nested/.env.production", "private.sqlite3",
              "nested/app.db-wal", "uploads/photo.png", "attachments/image.jpg", "backups/db.zip",
              "data/personal.json", "imports/raw.csv", "records.xlsx", "movie.mp4", "secret.pem",
              "shiny-backup.zip", "shiny-backup (1).zip", "shiny-data.json"]
    ignored = git("check-ignore", "--no-index", "-z", "--stdin", input="\0".join(probes) + "\0")
    require(set(ignored.stdout.rstrip("\0").split("\0")) == set(probes), "Sensitive path ignore probes failed")
    for safe in [".env.example", "docs/PRODUCT_SPEC_V1.1.md", "tests/fixtures/species.json"]:
        require(git("check-ignore", "--no-index", safe).returncode == 1, f"Overbroad ignore: {safe}")
    tracked_result = git("ls-files", "-z")
    require(tracked_result.returncode == 0, "Cannot enumerate tracked files")
    tracked = [p for p in tracked_result.stdout.split("\0") if p]
    ignored_tracked = git("check-ignore", "--no-index", "-z", "--stdin", input="\0".join(tracked) + "\0")
    require(not ignored_tracked.stdout.strip(), "Tracked files match sensitive/build ignore rules")
    secret_patterns = [rb"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----",
                       rb"gh[pousr]_[A-Za-z0-9]{30,}", rb"github_pat_[A-Za-z0-9_]{40,}"]
    for name in tracked:
        path = ROOT / name
        if path.is_file():
            content = path.read_bytes()
            require(not any(re.search(pattern, content) for pattern in secret_patterns),
                    f"Possible credential in tracked file {name}")

    # pull_request event is untrusted input; only validate fields, never execute them.
    if os.environ.get("GITHUB_EVENT_NAME") == "pull_request":
        event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text("utf-8"))
        pr = event["pull_request"]
        base, head = pr["base"]["ref"], pr["head"]["ref"]
        if base == "dev":
            match = re.fullmatch(r"feature/(T\d{3})-[a-z0-9]+(?:-[a-z0-9]+)*", head)
            require(match is not None and match.group(1) in expected_ids, "PR to dev needs feature/Txxx-name")
        elif base == "prod":
            require(head == "dev" and pr["head"]["repo"]["full_name"] == pr["base"]["repo"]["full_name"],
                    "Production PR must come from same-repository dev; release still needs user approval")
        else:
            ERRORS.append(f"Unexpected PR target {base}")


if __name__ == "__main__":
    try:
        validate()
    except (KeyError, ValueError, OSError) as exc:
        ERRORS.append(str(exc))
    if ERRORS:
        print("\n".join(f"ERROR: {e}" for e in ERRORS), file=sys.stderr)
        sys.exit(1)
    print("PASS: frozen documents; 29 tasks/definitions/dependencies/states; 16 ADRs; governance; ignore/credential checks; PR policy")
