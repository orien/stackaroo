import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Stackaroo',
  description: 'User documentation for Stackaroo.',
  lang: 'en-GB',
  appearance: 'auto',
  cleanUrls: true,
  lastUpdated: true,
  markdown: {
    lineNumbers: true
  },
  themeConfig: {
    nav: [],
    sidebar: [],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/orien/stackaroo' }
    ],
    footer: {
      message: 'Released under the BSD 3-Clause licence.',
      copyright: '© Stackaroo contributors'
    }
  }
})
