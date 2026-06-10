#!/usr/bin/env python3
"""Sync docs/tasks/tasks.json to GitHub Issues.

Reads the task queue, creates/updates/closes issues to match.
Run from repo root.
"""
import json, os, subprocess, sys
from pathlib import Path

REPO = os.environ.get("GITHUB_REPOSITORY") or subprocess.run(
    ["gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner"],
    capture_output=True, text=True, timeout=10
).stdout.strip()

TASKS_FILE = Path("docs/tasks/tasks.json")
MAP_FILE = Path(".github/task-issue-map.json")
GH = ["gh", "issue"]

PRIORITY_LABELS = {1: "P1", 2: "P2", 3: "P3"}

def load_json(path):
    if path.exists():
        try: return json.loads(path.read_text())
        except: pass
    return {}

def save_json(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2))

def issue_title(tid, title):
    return f"[{tid}] {title}"

def gh(*args):
    return subprocess.run(GH + list(args), capture_output=True, text=True, timeout=30)

def existing_issues():
    """Get all open issues with [TASK-ID] prefix."""
    r = gh("list", "--state", "all", "--json", "number,title,labels,state", "--limit", "100")
    if r.returncode != 0:
        print(f"  Error listing issues: {r.stderr[:200]}")
        return {}
    issues = {}
    for issue in json.loads(r.stdout):
        if issue["title"].startswith("["):
            issues[issue["number"]] = issue
    return issues

def main():
    if not TASKS_FILE.exists():
        print(f"No tasks.json at {TASKS_FILE}")
        sys.exit(0)

    tasks = load_json(TASKS_FILE).get("tasks", [])
    mapping = load_json(MAP_FILE)
    all_issues = existing_issues()

    # Inverse: issue_number → task_id from mapping
    issue_to_task = {v: k for k, v in mapping.items()}

    task_map = {t["id"]: t for t in tasks}
    changes = {"created": 0, "updated": 0, "closed": 0, "reopened": 0}

    # Sync each task
    for task in tasks:
        tid = task["id"]
        title = task.get("title", "")
        priority = task.get("priority", 3)
        status = task.get("status", "todo")
        description = task.get("description", "")
        labels = [PRIORITY_LABELS.get(priority, "P3")]

        if status == "completed":
            labels.append("status:completed")
        elif status == "todo":
            labels.append("status:todo")
        elif status == "in_progress":
            labels.append("status:in-progress")

        existing_issue_num = mapping.get(tid)

        if existing_issue_num:
            # Update existing issue
            issue = all_issues.get(existing_issue_num)
            if not issue:
                # Issue was deleted or mapping stale — recreate
                mapping.pop(tid, None)
                existing_issue_num = None

        if existing_issue_num:
            issue_state = all_issues.get(existing_issue_num, {}).get("state", "")

            if status == "completed" and issue_state != "closed":
                r = gh("close", str(existing_issue_num), "-m", f"Completed: {title}")
                if r.returncode == 0:
                    changes["closed"] += 1
                continue

            if status != "completed" and issue_state == "closed":
                r = gh("reopen", str(existing_issue_num))
                if r.returncode == 0:
                    changes["reopened"] += 1

            # Update labels and title
            label_str = ",".join(labels)
            r = gh("edit", str(existing_issue_num), "--title", issue_title(tid, title), "--add-label", f'"{label_str}"')
            if r.returncode == 0:
                changes["updated"] += 1

        else:
            if status == "completed":
                continue  # Don't create issue for already-completed tasks

            # Create new issue
            body = f"{description}\n\n---\n*Auto-synced from `docs/tasks/tasks.json` — task `{tid}`*"
            label_str = ",".join(labels)
            r = gh("create", "--title", issue_title(tid, title), "--label", f'"{label_str}"', "--body", body)
            if r.returncode == 0:
                num = int(r.stdout.strip().split("/")[-1])
                mapping[tid] = num
                changes["created"] += 1

    # Close issues for deleted tasks
    for tid, issue_num in list(mapping.items()):
        if tid not in task_map:
            r = gh("close", str(issue_num), "-m", "Task removed from queue")
            if r.returncode == 0:
                mapping.pop(tid)
                changes["closed"] += 1

    save_json(MAP_FILE, mapping)

    # Summary
    parts = [f"{k}={v}" for k, v in changes.items() if v > 0]
    print(f"Synced {len(tasks)} tasks: {', '.join(parts) if parts else 'no changes'}")


if __name__ == "__main__":
    main()
