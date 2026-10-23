"""Build a synthetic, real SQLite fixture; never open or modify the user's database."""

import json
import sqlite3
import sys
from pathlib import Path


fixture_dir = Path(sys.argv[1])
fixture_dir.mkdir(parents=True, exist_ok=True)
fixture_path = fixture_dir / "conversation_summaries.db"
if fixture_path.exists():
    fixture_path.unlink()

thread_id = "11111111-1111-4111-8111-111111111111"
with sqlite3.connect(fixture_path) as connection:
    # These column names and relative order match the inspected runtime schema.
    connection.execute(
        "CREATE TABLE conversation_summaries ("
        "conversation_id TEXT PRIMARY KEY, title TEXT NOT NULL, preview TEXT NOT NULL, "
        "step_count INTEGER NOT NULL, last_modified_time TEXT NOT NULL, "
        "workspace_uris TEXT NOT NULL, status TEXT NOT NULL)"
    )
    connection.execute(
        "INSERT INTO conversation_summaries VALUES (?, ?, ?, ?, ?, ?, ?)",
        (
            thread_id,
            "Synthetic project mapping fixture",
            "Command Working Directory: D:/UnrelatedPreview\n",
            1,
            "2026-10-18T00:00:00Z",
            json.dumps(["file:///C:/ActualProject"]),
            "IDLE",
        ),
    )

print("Synthetic SQLite fixture: conversation_id=" + thread_id)
print("Authoritative workspace_uris=" + json.dumps(["file:///C:/ActualProject"]))
print("Unrelated preview contains D:/UnrelatedPreview; it must not override workspace_uris.")
