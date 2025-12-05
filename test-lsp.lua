-- Minimal Neovim config for testing gitleaks-ls
-- Usage: nvim -u test-lsp.lua examples/test_file.go

-- Enable LSP logging
vim.lsp.set_log_level('DEBUG')

-- Better diagnostics display
vim.diagnostic.config({
  virtual_text = true,
  signs = true,
  underline = true,
  update_in_insert = false,
  severity_sort = true,
})

-- Create diagnostic signs
vim.fn.sign_define('DiagnosticSignError', { text = '✗', texthl = 'DiagnosticSignError' })
vim.fn.sign_define('DiagnosticSignWarn', { text = '⚠', texthl = 'DiagnosticSignWarn' })
vim.fn.sign_define('DiagnosticSignInfo', { text = 'ℹ', texthl = 'DiagnosticSignInfo' })
vim.fn.sign_define('DiagnosticSignHint', { text = '💡', texthl = 'DiagnosticSignHint' })

-- Track if LSP started
local lsp_started = false

-- Start gitleaks-ls
local function start_lsp()
  if lsp_started then return end
  lsp_started = true
  
  local server_path = vim.fn.getcwd() .. '/gitleaks-ls'
  
  -- Check if server exists
  if vim.fn.filereadable(server_path) == 0 then
    print('❌ ERROR: gitleaks-ls not found at: ' .. server_path)
    print('   Run: go build -o gitleaks-ls')
    return
  end
  
  print('🚀 Starting gitleaks-ls...')
  
  vim.lsp.start({
    name = 'gitleaks-ls',
    cmd = { server_path },
    root_dir = vim.fn.getcwd(),
    on_attach = function(client, bufnr)
      print('✅ LSP attached: ' .. client.name .. ' to buffer ' .. bufnr)
      
      -- Set up buffer-local keymaps
      local opts = { buffer = bufnr, silent = true }
      vim.keymap.set('n', 'K', vim.lsp.buf.hover, opts)
      vim.keymap.set('n', '<leader>ca', vim.lsp.buf.code_action, opts)
      vim.keymap.set('n', '[d', vim.diagnostic.goto_prev, opts)
      vim.keymap.set('n', ']d', vim.diagnostic.goto_next, opts)
      vim.keymap.set('n', '<leader>q', vim.diagnostic.setqflist, opts)
      
      print('📋 Keybindings set:')
      print('   K           - Hover documentation')
      print('   <leader>ca  - Code actions')
      print('   ]d / [d     - Next/Previous diagnostic')
      print('   <leader>q   - Quickfix list')
      
      -- Force a diagnostic refresh
      vim.schedule(function()
        vim.diagnostic.reset()
      end)
    end,
    on_exit = function(code, signal, client_id)
      print('⚠️  LSP server exited: code=' .. code)
    end,
  })
end

-- Start LSP on buffer read
vim.api.nvim_create_autocmd({'BufReadPost', 'BufNewFile'}, {
  callback = function()
    vim.defer_fn(start_lsp, 100)
  end,
})

-- Add debugging commands
vim.api.nvim_create_user_command('LspClients', function()
  local clients = vim.lsp.get_active_clients()
  if #clients == 0 then
    print('❌ No active LSP clients')
  else
    for _, client in ipairs(clients) do
      print('✅ Client: ' .. client.name .. ' (id: ' .. client.id .. ')')
      print('   Root: ' .. (client.config.root_dir or 'none'))
    end
  end
end, {})

vim.api.nvim_create_user_command('LspLog', function()
  vim.cmd('edit ' .. vim.lsp.get_log_path())
end, {})

vim.api.nvim_create_user_command('LspInfo', function()
  local clients = vim.lsp.get_active_clients({ bufnr = 0 })
  if #clients == 0 then
    print('❌ No LSP client attached to this buffer')
    print('   Try: :LspClients to see all clients')
    print('   Try: :LspLog to view server logs')
  else
    for _, client in ipairs(clients) do
      print('✅ Client: ' .. client.name)
      print('   Root: ' .. (client.config.root_dir or 'none'))
      local caps = client.server_capabilities
      print('   Capabilities:')
      print('     Hover: ' .. (caps.hoverProvider and 'yes' or 'no'))
      print('     CodeAction: ' .. (caps.codeActionProvider and 'yes' or 'no'))
      print('     TextSync: ' .. (caps.textDocumentSync and 'yes' or 'no'))
    end
  end
end, {})

vim.api.nvim_create_user_command('DiagShow', function()
  local diags = vim.diagnostic.get(0)
  if #diags == 0 then
    print('No diagnostics in current buffer')
    print('Try saving the file with :w')
  else
    print('📋 Diagnostics (' .. #diags .. '):')
    for i, d in ipairs(diags) do
      print(string.format('  %d. Line %d: %s', i, d.lnum + 1, d.message))
    end
  end
end, {})

-- Print startup message
print('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━')
print('🔍 Gitleaks LSP Test Config')
print('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━')
print('Commands:')
print('  :LspInfo     - Check LSP status')
print('  :LspClients  - List active clients')
print('  :LspLog      - Open LSP log')
print('  :DiagShow    - Show diagnostics')
print('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━')
