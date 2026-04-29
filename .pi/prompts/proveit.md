---
description: Generate an HTML proof report with real screenshots showing a feature works
argument-hint: "[feature or instructions]"
---
Use the `proveit` skill to generate an HTML proof report with real browser screenshots and verification evidence for this implementation.

User instructions/context:
$ARGUMENTS

Follow the skill workflow: run relevant tests/builds, run the real app if applicable, seed realistic data through real routes or supported setup flows, capture screenshots for important states, and write the report under `designs/<feature>-proof-report/index.html` unless a better path is obvious. Return the report path, screenshots captured, and verification commands that passed.
