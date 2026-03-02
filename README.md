# wo

`wo` is a fast workspace manager for repo-style directories.

## What It Does

- Jump to a workspace by repo name: `wo harp`
- Browse all indexed workspaces in a TUI: `wo`
- Resolve naming conflicts with an interactive picker
- Discover workspaces from `.git` directories or `.wo` files
- Run trusted enter hooks (for example `nvim .`) after entering

## Install

```bash
make install
# or
./scripts/install.sh
```

## Uninstall

```bash
make uninstall
# or
./scripts/uninstall.sh
```

`uninstall` removes:
- installed `wo` binary
- installed `wo` man page
- runtime SQLite index/trust state (`wo.db*`)

`uninstall` keeps:
- workspace `.wo` files
- config files

Then enable shell integration:

```bash
# zsh
source <(wo init zsh)

# bash
source <(wo init bash)

# fish
wo init fish | source
```

After loading shell init, `wo <TAB>` completes workspace names from your index.

## Basic Usage

```bash
# index default roots (~/workspaces if present)
wo scan --depth 1

#recursively scan for .git/ or .wo files (if your project isn't tracked)
wo scan --root your/custom/paths/here --depth 2

# jump by repo name
wo harp

# browse all projects grouped by owner
wo

# skip enter hooks for this jump
wo harp --clean

# force picker even if one match exists
wo harp --pick
```

## Workspace Detection

A directory is indexed if it has either:

- `.git/`
- `.wo`

### `.wo` file format (TOML)

```toml
name = "harp"
owner = "hackutd"

[enter]
commands = ["nvim ."]
shell = "inherit"
```

## Config

Global config path:

- `~/.config/wo/config.toml`

Default config values:

```toml
roots = ["~/workspaces"]

[scan]
depth_default = 1
follow_symlink = false

[search]
backend = "auto" # auto|internal|fzf

[ui]
theme = "gh"

[hooks]
enabled = true

[correction]
enabled = true
min_score = 0.72
min_gap = 0.10
```

## Commands

- `wo <query> [--clean] [--pick]`
- `wo`
- `wo scan [--root <path>] [--depth <n>] [--follow-symlinks] [--prune]`
- `wo list [--owner <owner>] [--json]`
- `wo doctor`
- `wo trust list|allow|deny|reset`
- `wo completion <bash|zsh|fish>`

## Notes

- Use `wo trust` to inspect and manage hook trust decisions.
- `fzf` is optional. If installed and `search.backend=auto|fzf`, wo uses it for interactive picking.
- `WO_DEBUG=1` enables debug logs.
