import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'MCPX',
  description: 'Secure gateway for MCP servers — from CLI to production',
  base: '/mcpx/',
  head: [
    ['meta', { name: 'theme-color', content: '#000000' }],
    ['meta', { property: 'og:title', content: 'MCPX — Secure MCP Gateway' }],
    ['meta', { property: 'og:description', content: 'Secure gateway for MCP servers — from CLI to production. Security policies, audit logging, lifecycle hooks, and monorepo workspaces.' }],
  ],

  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'MCPX',

    nav: [
      { text: 'Guide', link: '/getting-started/installation' },
      { text: 'Security', link: '/security/overview' },
      { text: 'Reference', link: '/reference/cli' },
      { text: 'Integrations', link: '/integrations/serena' },
      {
        text: 'v1.4.0',
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
          text: 'Core Concepts',
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
      '/security/': [
        {
          text: 'Security',
          items: [
            { text: 'Overview', link: '/security/overview' },
            { text: 'Policies', link: '/security/policies' },
            { text: 'Modes', link: '/security/modes' },
            { text: 'Audit Logging', link: '/security/audit-logging' },
            { text: 'Examples', link: '/security/examples' },
          ],
        },
      ],
      '/workspaces/': [
        {
          text: 'Workspaces',
          items: [
            { text: 'Overview', link: '/workspaces/overview' },
            { text: 'Configuration', link: '/workspaces/configuration' },
            { text: 'Serena Monorepo', link: '/workspaces/serena-monorepo' },
          ],
        },
      ],
      '/integrations/': [
        {
          text: 'Integrations',
          items: [
            { text: 'Serena', link: '/integrations/serena' },
            { text: 'Databases', link: '/integrations/databases' },
            { text: 'Project Management', link: '/integrations/project-management' },
            { text: 'Communication', link: '/integrations/communication' },
            { text: 'CI/CD Pipelines', link: '/integrations/ci-cd' },
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
          text: 'Examples (Legacy)',
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
