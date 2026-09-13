#!/usr/bin/env python3
"""Verify MESH-107 sources resolve and the mechanism map is complete.

This is the measurement step for a research-loop task: another agent should
be able to rerun this file and see which citations still exist and whether
every adopted idea still names a prediction and a falsification.
"""

from __future__ import annotations

import json
import ssl
import sys
import urllib.error
import urllib.request
from pathlib import Path

HERE = Path(__file__).resolve().parent
SOURCES = HERE / "sources.json"
MAP = HERE / "mechanism-map.json"
OUT = HERE / "verification.json"

REQUIRED_ADOPTED = (
    "id",
    "title",
    "sources",
    "mechanism",
    "software_hypothesis",
    "measurable_prediction",
    "experiment_task",
    "falsification",
    "why_analogy_may_fail",
)

UA = "MeshyAnts-MESH-107-source-check/1.0"


def fetch_status(url: str, timeout: float = 20.0) -> dict:
    ctx = ssl.create_default_context()
    req = urllib.request.Request(
        url,
        method="GET",
        headers={"User-Agent": UA, "Accept": "text/html,application/json,*/*"},
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout, context=ctx) as resp:
            return {
                "url": url,
                "ok": 200 <= resp.status < 400,
                "status": resp.status,
                "final_url": resp.geturl(),
            }
    except urllib.error.HTTPError as e:
        # DOI/publisher stacks often 403 bots on GET but the identifier still exists.
        ok = e.code in {401, 403, 429} or 200 <= e.code < 400
        return {
            "url": url,
            "ok": ok,
            "status": e.code,
            "final_url": getattr(e, "url", url),
            "note": "http_error_treated_as_identifier_exists"
            if e.code in {401, 403, 429}
            else None,
        }
    except Exception as e:  # noqa: BLE001 — want every fetch failure in the record
        return {"url": url, "ok": False, "status": None, "error": type(e).__name__, "detail": str(e)}


def main() -> int:
    sources_doc = json.loads(SOURCES.read_text())
    mapping = json.loads(MAP.read_text())

    map_errors: list[str] = []
    for idea in mapping.get("adopted", []):
        missing = [k for k in REQUIRED_ADOPTED if not idea.get(k)]
        if missing:
            map_errors.append(f"{idea.get('id', '?')}: missing {missing}")
        pred = str(idea.get("measurable_prediction", ""))
        if len(pred) < 40:
            map_errors.append(f"{idea.get('id', '?')}: measurable_prediction too thin")
        if "podcast" in pred.lower():
            map_errors.append(f"{idea.get('id', '?')}: prediction cites a podcast")

    if not mapping.get("rejected_translations"):
        map_errors.append("rejected_translations empty")
    if not mapping.get("falsification_of_watch"):
        map_errors.append("watch-level falsification missing")

    known = {s["id"] for s in sources_doc.get("sources", [])}
    for idea in mapping.get("adopted", []):
        for sid in idea.get("sources", []):
            if sid not in known:
                map_errors.append(f"{idea['id']}: unknown source {sid}")

    fetches = []
    for src in sources_doc.get("sources", []):
        url = src.get("url")
        if not url:
            fetches.append({"id": src["id"], "ok": False, "error": "no_url"})
            continue
        result = fetch_status(url)
        result["id"] = src["id"]
        fetches.append(result)

    rejected = sources_doc.get("rejected_as_evidence", [])
    used_podcast = any(
        "podcast" in json.dumps(idea).lower() and "not" not in json.dumps(idea).lower()
        for idea in mapping.get("adopted", [])
    )

    record = {
        "task_id": "MESH-107",
        "sources_checked": len(fetches),
        "sources_ok": sum(1 for f in fetches if f.get("ok")),
        "map_errors": map_errors,
        "adopted_count": len(mapping.get("adopted", [])),
        "rejected_translation_count": len(mapping.get("rejected_translations", [])),
        "rejected_as_evidence_count": len(rejected),
        "podcast_used_as_evidence": used_podcast,
        "fetches": fetches,
    }
    OUT.write_text(json.dumps(record, indent=2) + "\n")

    failed_fetches = [f for f in fetches if not f.get("ok")]
    print(json.dumps({k: record[k] for k in record if k != "fetches"}, indent=2))
    if failed_fetches:
        print("failed_fetches:", json.dumps(failed_fetches, indent=2))
    if map_errors or failed_fetches or used_podcast:
        return 1
    print("MESH-107 source+map verification: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
