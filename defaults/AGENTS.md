# Miya Workspace

This is the default workspace for Miya conversations.

- Treat files in this directory as user-owned working data.
- Read relevant files before editing them and preserve unrelated content.
- Use the available skills for Miya Desktop and `~/.miya/config.json` configuration tasks.
- Keep credentials and tokens out of workspace files unless the user explicitly requests otherwise.
- Store generated files here when the user does not provide another destination.

Miya runtime data lives under `~/.miya/`. The shared configuration is `~/.miya/config.json`, installed skills are under `~/.miya/skills`, sessions are under `~/.miya/sessions`, and logs are under `~/.miya/logs`.
