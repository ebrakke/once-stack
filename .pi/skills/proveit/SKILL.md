---
name: proveit
description: Generate an HTML proof report with real browser screenshots and verification evidence showing that a web UI feature or implementation actually works. Use this whenever the user says "prove it", asks for screenshots, asks for an HTML report validating a feature, wants evidence that a UI change was implemented correctly, or invokes /proveit.
compatibility: Requires bash, Python 3, and a headless browser such as chromium/google-chrome for screenshot capture.
---

# Prove It

Use this skill to produce a reviewable HTML report proving that a feature works, especially server-rendered web UI changes. The report should include real screenshots from a running app, commands/tests that were run, and concise implementation evidence.

## Core workflow

1. **Understand what needs proving**
   - Identify the feature/change under review.
   - Identify how to run the app locally.
   - Identify the important states/screens to prove.
   - If necessary, create realistic test data through the real app/API, not by faking screenshots.

2. **Run verification commands**
   - Run appropriate tests first, usually `go test ./...` or the project quality command.
   - If generated assets are involved, run the asset build command first, e.g. `just css-build`.
   - Capture both the commands and whether they passed for the report.

3. **Run the app and capture screenshots**
   - Start the real app locally on an explicit port.
   - Wait for a health endpoint or page to respond.
   - Seed data through real HTTP routes, CLI commands, or app-supported setup flows.
   - Capture screenshots with Chromium/headless browser for all relevant states.

4. **Collect implementation evidence**
   Include a few concrete facts, such as:
   - Which shared assets/templates/routes are used.
   - Which old implementation was removed.
   - Which CSS/classes/scripts are present or absent in rendered output.
   - Which tests passed.

5. **Generate an HTML report**
   - Put it in a stable review location, usually `designs/<feature>-proof-report/index.html`.
   - Store screenshots under `designs/<feature>-proof-report/screenshots/`.
   - The report should be self-contained except for relative screenshot image files.
   - Include enough context that the user can review the evidence without reading the terminal transcript.

## Preferred helper script

This skill includes `scripts/proveit.py`, which can launch an app, run setup commands, capture screenshots, and write a report.

Example:

```bash
python .pi/skills/proveit/scripts/proveit.py \
  --title "Notes UI Migration Proof" \
  --summary "Notes renders through the shared ONCE UI system." \
  --out designs/notes-ui-proof \
  --start-command 'PORT=18080 STORAGE_DIR=$(mktemp -d) go run ./cmd/notes' \
  --base-url http://127.0.0.1:18080 \
  --health-path /up \
  --verify-command 'go test ./...' \
  --setup-command 'curl -fsS -X POST "$BASE_URL/notes" -H "Content-Type: application/x-www-form-urlencoded" --data-urlencode "title=Demo note" --data-urlencode "body=## Hello" -D "$REPORT_DIR/create.headers" -o /dev/null' \
  --screenshot 'Notes list|/|List page with shared cards and primary button' \
  --screenshot 'New note|/new|Create form with shared form controls' \
  --evidence 'Shared CSS|The app links /assets/once/once.css' \
  --implementation-note 'Notes routes mount ui.AssetsHandler at /assets/once/'
```

The script exposes these environment variables to setup and verify commands:

- `BASE_URL` — base URL of the running app.
- `REPORT_DIR` — output directory for logs/intermediate files.
- `SCREENSHOT_DIR` — screenshot output directory.

## When the helper script is not enough

If the app requires a custom login flow, unusual setup, or dynamic URLs that are easier to handle manually, follow the same workflow with direct bash commands and then write the HTML report yourself. Keep the report structure below.

## Report structure

Use this structure unless the user asks for something else:

1. Title and short summary.
2. Verification checklist.
3. Commands run and pass/fail status.
4. Implementation evidence.
5. Screenshot gallery with captions for each state.
6. Notes/limitations, if any.

## Screenshot guidance

- Use the real running app.
- Prefer deterministic sample data.
- Capture multiple states, not only the happy-path screen.
- Include error/empty states when relevant.
- Use a consistent viewport, usually `1280x1000` for desktop UI.
- Do not fabricate screenshots or use static mockups as proof.

## Good proof examples

- For a notes app UI migration: list, detail, edit form, new form, error state, rendered CSS checks.
- For an upload feature: empty state, upload form, successful uploaded file row, download route, validation error.
- For auth: logged-out page, login form, logged-in dashboard, unauthorized error, cookie/session evidence.

## Final response

Tell the user:

- Where the report was written.
- Which screenshots were captured.
- Which verification commands passed.
- Any limitations or follow-up checks worth doing.
