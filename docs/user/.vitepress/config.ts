import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Stackaroo',
  description: 'User documentation for Stackaroo.',
  base: '/stackaroo/',
  lang: 'en-GB',
  appearance: 'auto',
  cleanUrls: true,
  lastUpdated: true,
  markdown: {
    lineNumbers: true
  },
  themeConfig: {
    nav: [
      { text: '🎓 Tutorials', link: '/tutorials/' },
      { text: '🔧 How-to Guides', link: '/how-to/' },
      { text: '💡 Explanations', link: '/explanation/' },
      { text: '📘 Reference', link: '/reference/' }
    ],
    sidebar: {
      '/tutorials/': [
        {
          text: '🎓 Tutorials',
          items: [
            { text: 'Overview', link: '/tutorials/' },
            { text: 'First Stack Deployment', link: '/tutorials/first-stack-deployment' }
          ]
        }
      ],
      '/how-to/': [
        {
          text: '🔧 How-to Guides',
          items: [
            { text: 'Overview', link: '/how-to/' },
            { text: 'Configure Stacks', link: '/how-to/configure-stacks' }
          ]
        }
      ],
      '/explanation/': [
        {
          text: '💡 Explanations',
          items: [{ text: 'Overview', link: '/explanation/' }]
        }
      ],
      '/reference/': [
        {
          text: '📘 Reference',
          items: [{ text: 'Overview', link: '/reference/' }]
        }
      ]
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/orien/stackaroo' }
    ],
    footer: {
      message: 'Released under the BSD 3-Clause licence.',
      copyright: '© Stackaroo contributors'
    }
  }
})
