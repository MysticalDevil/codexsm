# codex-sm

`codex-sm` 是一个本地 Codex Session 管理工具（Go 实现），支持：

- 会话列表查看（`list`）
- 安全删除（`delete`，默认 dry-run）
- 会话分组统计（`group`，按 `day` / `health`）

## Features

- 默认安全：`delete` 默认 `--dry-run=true`
- 真实删除需要显式确认：`--dry-run=false --confirm`
- 默认软删除：移动到 `~/.codex/trash/sessions/...`
- 可选硬删除：`--hard`
- 交互确认：真实删除默认会提示确认（可 `--interactive-confirm=false`）
- 输出优化：默认显示最近 10 条，支持 `--detailed`、`--pager`、`--color`

## Install / Build

```bash
make build
```

或：

```bash
go build ./cmd/csm
```

## Quick Start

```bash
# 列出最近 10 条
./csm list

# 详细模式
./csm list --detailed

# 分页查看全部
./csm list --limit 0 --pager

# 按天分组
./csm group --by day

# 按健康状态分组
./csm group --by health

# 删除预演（默认 dry-run）
./csm delete --id-prefix 019ca9

# 真实软删除（需确认）
./csm delete --id-prefix 019ca9 --dry-run=false --confirm
```

## Development

```bash
make fmt
make lint
make test
make check
```

说明：

- `fmt` 使用 `gofumpt`
- `lint` 使用 `go vet`

## License

BSD 3-Clause License，详见 [LICENSE](./LICENSE)。
