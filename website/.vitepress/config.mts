import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'MCPX',
  description: 'MCP servers as CLI tools — built for AI agents',
  base: '/mcpx/',
  head: [
    ['meta', { name: 'theme-color', content: '#000000' }],
    ['meta', { property: 'og:title', content: 'MCPX' }],
    ['meta', { property: 'og:description', content: 'MCP servers as CLI tools — built for AI agents' }],
  ],

  themeConfig: {
    logo: false,
    siteTitle: 'MCPX',

    nav: [
      { text: 'Guide', link: '/getting-started/installation' },
      { text: 'Reference', link: '/reference/cli' },
      { text: 'Troubleshooting', link: '/troubleshooting/' },
      { text: 'Examples', link: '/examples/serena' },
      {
        text: 'v1.0.0',
        items: [
          { text: 'Changelog', link: '/about/changelog' },
          { text: 'GitHub', link: 'https://github.com/codestz/mcpx' },
        ],
      },
    ],

    sidebar: {
      '/getting-started/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Installation', link: '/getting-started/installation' },
            { text: 'Quick Start', link: '/getting-started/quick-start' },
            { text: 'AI Agent Setup', link: '/getting-started/ai-agent-setup' },
          ],
        },
      ],
      '/guide/': [
        {
          text: 'Guide',
          items: [
            { text: 'Configuration', link: '/guide/configuration' },
            { text: 'Dynamic Variables', link: '/guide/variables' },
            { text: 'Secrets', link: '/guide/secrets' },
            { text: 'Daemon Mode', link: '/guide/daemon-mode' },
            { text: 'Output Modes', link: '/guide/output-modes' },
            { text: 'Shell Completion', link: '/guide/shell-completion' },
          ],
        },
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'CLI Commands', link: '/reference/cli' },
            { text: 'Config Schema', link: '/reference/config-schema' },
            { text: 'Exit Codes', link: '/reference/exit-codes' },
            { text: 'Environment', link: '/reference/environment' },
          ],
        },
      ],
      '/troubleshooting/': [
        {
          text: 'Troubleshooting',
          items: [
            { text: 'Overview', link: '/troubleshooting/' },
            { text: 'Connection Errors', link: '/troubleshooting/connection-errors' },
            { text: 'Daemon Issues', link: '/troubleshooting/daemon-issues' },
            { text: 'Secrets Errors', link: '/troubleshooting/secrets-errors' },
            { text: 'Config Errors', link: '/troubleshooting/config-errors' },
            { text: 'Platform-Specific', link: '/troubleshooting/platform-specific' },
          ],
        },
      ],
      '/examples/': [
        {
          text: 'Examples',
          items: [
            { text: 'Serena', link: '/examples/serena' },
            { text: 'Multi-Server Project', link: '/examples/multi-server' },
            { text: 'CI/CD Pipelines', link: '/examples/ci-cd' },
          ],
        },
      ],
      '/about/': [
        {
          text: 'About',
          items: [
            { text: 'How It Works', link: '/about/how-it-works' },
            { text: 'Why MCPX', link: '/about/why-mcpx' },
            { text: 'Comparison', link: '/about/comparison' },
            { text: 'Changelog', link: '/about/changelog' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/codestz/mcpx' },
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright 2025 codestz',
    },

    search: {
      provider: 'local',
    },

    outline: {
      level: [2, 3],
    },
  },
})
