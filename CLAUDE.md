# CLAUDE.md — Правила проекта WinSettingsGui

## Bash Safety

- Never chain commands with `&&`, `;`, or `||`. One command per tool call.
- Never use command substitution (backticks, `$()`). Run the inner command first, use the result in the next call.
- Never use process substitution (`<()`, `>()`). Write to temp files instead.
- Never start a command with `cd`. Use `git -C <path>` or absolute paths.
- Never use shell redirects to create files (`echo > file`, `cat << EOF > file`). Use the Write tool.
- Never use `sed -i` for in-place file edits. Use the Edit tool.
- Never use `export`. Use inline prefixing: `PATH=/opt/homebrew/bin:$PATH command`.
- Never create LaunchAgents or run `launchctl`. Write the plist file and tell the user to load it manually.

## Рабочий процесс

После выполнения любой задачи, если были изменены или созданы файлы:
- После каждой доработки обновляй `README.md` — он должен всегда отражать актуальное состояние проекта

## Исследование проекта

При необходимости понять структуру, возможности или API проекта — **сначала читай `README.md`**, он содержит актуальное описание всего проекта.

## Технологический стек

- Язык: **GO**

## Чего не делать

- Не добавлять избыточные комментарии и docstring к коду, который не менялся
