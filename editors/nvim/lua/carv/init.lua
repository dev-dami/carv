local M = {}

--- Setup the Carv LSP client.
--- Call from your Neovim config:
---   require("carv").setup({})
function M.setup(opts)
  opts = opts or {}

  local bin = vim.fn.executable("carv")
  if bin == 0 then
    vim.notify("carv: 'carv' binary not found in $PATH", vim.log.levels.WARN)
    return
  end

  local lsp_opts = {
    name = "carv-lsp",
    cmd = { "carv", "lsp" },
    root_dir = vim.fs.dirname(vim.fs.find({ "carv.toml", ".git" }, { upward = true })[1]) or vim.fn.getcwd(),
    single_file_support = true,
  }

  lsp_opts = vim.tbl_deep_extend("force", lsp_opts, opts.lsp or {})

  vim.api.nvim_create_autocmd("FileType", {
    pattern = "carv",
    callback = function()
      vim.lsp.start(lsp_opts)
    end,
  })
end

return M
