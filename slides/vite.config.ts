import { defineConfig } from 'vite'

export default defineConfig({
  optimizeDeps: {
    // The WASM lexer bundled in Vite 8 accumulates memory across files and
    // exceeds its 16MB cap when scanning Slidev's large deps (typescript,
    // mermaid, monaco). `disabled: true` is the only path that bypasses
    // extractExportsData entirely — noDiscovery alone doesn't work because
    // ViteSlidevPlugin re-adds an `include` list that re-enables the scan.
    disabled: true,
  },
})
