"""M1 black-box acceptance: only a newly owned disposable SQLite database."""

from contextlib import contextmanager
import http.client
import json
import os
from pathlib import Path
import signal
import socket
import sqlite3
import subprocess
import tempfile
import time
from urllib.parse import urlencode

ROOT = Path(__file__).resolve().parents[1]


@contextmanager
def server(db_path):
    with socket.socket() as probe:
        probe.bind(("127.0.0.1", 0))
        port = probe.getsockname()[1]
    process = subprocess.Popen(
        [str(ROOT / "bin/shiny-tracker")], cwd=ROOT,
        env=dict(os.environ, SHINY_HTTP_PORT=str(port), SHINY_DB_PATH=str(db_path)),
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
    )

    def call(method, path, data=None, status=200):
        connection = http.client.HTTPConnection("127.0.0.1", port, timeout=2)
        try:
            body = json.dumps(data) if data is not None else None
            connection.request(method, "/api/v1" + path, body,
                               {"Content-Type": "application/json"} if body else {})
            response = connection.getresponse()
            raw = response.read()
            result = json.loads(raw) if raw else None
            assert response.status == status, (method, path, response.status, result)
            return result
        finally:
            connection.close()

    try:
        deadline = time.monotonic() + 10
        while True:
            if process.poll() is not None:
                raise AssertionError("backend exited before readiness")
            try:
                call("GET", "/health")
                break
            except OSError:
                if time.monotonic() >= deadline:
                    raise AssertionError("backend readiness timed out")
                time.sleep(0.05)
        yield call
    finally:
        if process.poll() is None:
            process.send_signal(signal.SIGTERM)
        try:
            output, _ = process.communicate(timeout=7)
        except subprocess.TimeoutExpired:
            process.kill()
            process.communicate()
            raise
        assert process.returncode == 0, (process.returncode, output)


def main():
    with tempfile.TemporaryDirectory(prefix="pst-m1-") as directory:
        db_path = Path(directory) / "acceptance.sqlite3"
        assert not db_path.exists()
        with server(db_path) as call:
            species = call("GET", "/reference/species")["data"]
            games = call("GET", "/reference/games")["data"]
            assert len(species) == 3 and len(games) == 4
            assert call("GET", "/reference/coverage")["data"]["coverage"] == "partial"
            first, second = species[0]["id"], species[1]["id"]
            original_game, current_game = games[0]["id"], games[1]["id"]
            independent = dict(current_species_id=first, original_species_id=first,
                               original_game_id=original_game, source="trade",
                               holding_status="held", obtained_date="2018")
            individual = call("POST", "/shiny-records", independent, 201)["data"]
            assert individual["hunt_id"] is None
            hunt_input = dict(species_id=first, status="in_progress", encounter_count=0,
                              count_unit="encounters", duration_seconds=0)
            hunt = call("POST", "/hunts", hunt_input, 201)["data"]
            hid = hunt["id"]
            call("PUT", f"/hunts/{hid}", dict(hunt_input, status="completed"), 409)
            outcome = dict(current_species_id=second, hunt_id=hid, source="self_hunted",
                           holding_status="held")
            a = call("POST", "/shiny-records", outcome, 201)["data"]["id"]
            b = call("POST", "/shiny-records", outcome, 201)["data"]["id"]
            assert a != b
            call("PUT", f"/hunts/{hid}", dict(hunt_input, status="completed"))
            call("DELETE", f"/hunts/{hid}", status=409)
            evolved = dict(independent, current_species_id=second, current_game_id=current_game,
                           current_location="Box 1", change_type="evolution")
            iid = individual["id"]
            updated = call("PUT", f"/shiny-records/{iid}", evolved)["data"]
            assert updated["id"] == iid and updated["original_species_id"] == first
            assert updated["original_game_id"] == original_game
            evolved.pop("change_type")
            evolved["holding_status"] = "traded"
            call("PUT", f"/shiny-records/{iid}", evolved)
            kinds = {event["kind"] for event in call("GET", f"/shiny-records/{iid}/history")["data"]}
            assert {"registered", "evolution", "location_change", "holding_change"} <= kinds
            query = urlencode(dict(original_species_id=first, current_species_id=second,
                                   original_game_id=original_game, current_game_id=current_game,
                                   has_hunt="false", holding_status="traded"))
            result = call("GET", "/shiny-records?" + query)
            assert result["meta"]["total"] == 1 and result["data"][0]["id"] == iid
            pages = [call("GET", f"/shiny-records?page={page}&page_size=1&sort=id&order=asc")
                     for page in (1, 2, 3, 4)]
            assert all(page["meta"]["total"] == 3 for page in pages)
            assert [page["data"][0]["id"] for page in pages[:3]] == [iid, a, b]
            assert pages[3]["data"] == []
            for path in ("/hunts?page=0", "/hunts?sort=encounter_count",
                         "/shiny-records?current_species_id=0", "/shiny-records?source=bad"):
                call("GET", path, status=400)
            call("DELETE", f"/shiny-records/{a}", status=204)
            call("DELETE", f"/shiny-records/{b}", status=409)
            call("PUT", f"/hunts/{hid}", dict(hunt_input, status="abandoned"))
            call("DELETE", f"/shiny-records/{b}", status=204)
            call("DELETE", f"/shiny-records/{iid}", status=204)
            assert call("GET", f"/shiny-records/{iid}/history")["meta"]["record_deleted"]
        # Reopen the same private test database: startup/query do not call a source.
        with server(db_path) as call:
            assert len(call("GET", "/reference/species")["data"]) == 3
            assert call("GET", f"/hunts/{hid}")["data"]["status"] == "abandoned"
            assert call("GET", "/shiny-records")["meta"]["total"] == 0
            assert call("GET", f"/shiny-records/{iid}/history")["meta"]["record_deleted"]
        with sqlite3.connect(f"file:{db_path}?mode=ro", uri=True) as db:
            assert db.execute("PRAGMA user_version").fetchone()[0] == 3
            assert db.execute("PRAGMA foreign_key_check").fetchall() == []
            assert db.execute("SELECT count(*) FROM shiny_records").fetchone()[0] == 3
            assert db.execute("SELECT count(*) FROM shiny_histories").fetchone()[0] >= 9
        print("PASS M1: empty SQLite init; offline seed; independent/0..N outcomes; CRUD/history; "
              "original/current filters; stable pages; bad inputs; last-outcome protection; "
              "restart persistence; foreign keys and retained identities")


if __name__ == "__main__":
    main()
