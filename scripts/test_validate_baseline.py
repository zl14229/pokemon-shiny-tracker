"""Negative regression checks for the governance validator (no business tests)."""

import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parents[1]


class BaselineValidationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name in ["AGENTS.md", "README_BASELINE.md", ".gitignore", ".gitattributes", "docs", ".github"]:
            src, dst = SOURCE / name, self.root / name
            if src.is_dir():
                shutil.copytree(src, dst)
            else:
                shutil.copyfile(src, dst)
        subprocess.run(["git", "init", "-q", str(self.root)], check=True)
        subprocess.run(["git", "-C", str(self.root), "add", "."], check=True)
        spec = importlib.util.spec_from_file_location("baseline", SOURCE / "scripts/validate_baseline.py")
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)
        self.module.ROOT = self.root
        self.module.os.environ = dict(self.module.os.environ, GITHUB_EVENT_NAME="local-test")

    def registry(self, change):
        path = self.root / "docs/tasks/status.json"
        data = json.loads(path.read_text("utf-8"))
        change(data["tasks"])
        path.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")

    def errors(self):
        self.module.validate()
        return "\n".join(self.module.ERRORS)

    def test_valid_baseline(self):
        self.assertEqual(self.errors(), "")

    def test_frozen_document_change_is_rejected(self):
        path = self.root / "docs/PRODUCT_SPEC_V1.1.md"
        path.write_text(path.read_text("utf-8") + "changed\n", encoding="utf-8")
        self.assertIn("Frozen document changed", self.errors())

    def test_dependency_drift_is_rejected(self):
        self.registry(lambda tasks: tasks[1].update(dependencies=[]))
        self.assertIn("T001: baseline dependencies mismatch", self.errors())

    def test_acceptance_drift_is_rejected(self):
        self.registry(lambda tasks: tasks[0].update(acceptance="different"))
        self.assertIn("T000: baseline acceptance mismatch", self.errors())

    def test_false_verified_state_is_rejected(self):
        self.registry(lambda tasks: tasks[0].update(status="verified", evidence={}))
        self.assertIn("T000: invalid merge_sha", self.errors())

    def test_forced_personal_data_is_rejected(self):
        (self.root / "personal.sqlite3").write_bytes(b"fake fixture")
        subprocess.run(["git", "-C", str(self.root), "add", "-f", "personal.sqlite3"], check=True)
        self.assertIn("Tracked files match", self.errors())

    def test_invalid_pr_branch_is_rejected(self):
        event = {"pull_request": {"base": {"ref": "dev"}, "head": {"ref": "task/TASK-000-old"}}}
        path = self.root / "event.json"
        path.write_text(json.dumps(event), encoding="utf-8")
        self.module.os.environ = dict(self.module.os.environ, GITHUB_EVENT_NAME="pull_request", GITHUB_EVENT_PATH=str(path))
        self.assertIn("PR to dev needs feature/Txxx-name", self.errors())


if __name__ == "__main__":
    unittest.main()
