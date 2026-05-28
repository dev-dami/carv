# Carv Neovim Support

Syntax highlighting and LSP integration for [Carv](https://github.com/dev-dami/carv) — a memory-safe language for embedded systems.

## Features

- **Syntax highlighting** for `.carv` files
- **Filetype detection** — automatically sets `filetype=carv`
- **LSP integration** — hooks up the `carv lsp` server via Neovim's built-in LSP client

## Installation

### lazy.nvim

```lua
{
  dir = "/path/to/carv/editors/nvim",
  opts = {},
}
```

Or copy the plugin into your Neovim config:

```bash
cp -r editors/nvim ~/.config/nvim/
```

## Usage

The LSP client starts automatically when you open a `.carv` file (requires the `carv` binary in `$PATH`).

Configure with:

```lua
require("carv").setup({
  lsp = {
    -- additional LSP client options
    -- e.g. on_attach, capabilities, etc.
  },
})
```
