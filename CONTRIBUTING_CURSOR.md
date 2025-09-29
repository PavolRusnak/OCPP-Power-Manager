# Contributing with Cursor

## Rules

- Always read `cursor_steps.json`. Work only on the item with `"status":"todo"` and lowest `step`.
- After completing a step, set its status to `"done"`, add any new artifacts, and increment `"next"` in the previous item if needed.
- Do not rename files or folders without updating `cursor_steps.json`.
- Apply changes without asking. Overwrite files if they exist.
- After each completed step, output a short changelog.
