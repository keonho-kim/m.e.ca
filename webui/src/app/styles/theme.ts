// Purpose: Define the monochrome glass material through Astryx's public theme contract.
// Pattern: Extension of the official neutral theme.
// Usage: The build compiles this descriptor; main imports its generated CSS and module.
// Related: webui/src/app/main.tsx, webui/src/shared/ui/icons/index.tsx, webui/DESIGN.md.

import { defineTheme } from '@astryxdesign/core/theme';
import { neutralTheme } from '@astryxdesign/theme-neutral/built';
import { mecaIcons } from '@/shared/ui/icons';

const font = 'system-ui, -apple-system, BlinkMacSystemFont, sans-serif';
export const mecaTheme = defineTheme({
  name: 'meca',
  extends: neutralTheme,
  icons: mecaIcons,
  typography: {
    body: { family: 'system-ui', fallbacks: font },
    heading: { family: 'system-ui', fallbacks: font },
  },
  motion: { fast: 130, medium: 220, slow: 360, ratio: 0.8 },
  tokens: {
    '--font-family-body': font,
    '--font-family-heading': font,
    '--color-background-body': ['#f5f5f5', '#0b0b0b'],
    '--color-background-surface': ['#ffffff', '#141414'],
    '--color-background-card': ['#ffffff', '#191919'],
    '--color-background-popover': ['rgb(255 255 255 / 88%)', 'rgb(28 28 28 / 92%)'],
    '--color-background-muted': ['#f5f5f5', '#202020'],
    '--color-accent': ['#171717', '#f5f5f5'],
    '--color-on-accent': ['#ffffff', '#171717'],
    '--color-accent-muted': ['rgb(0 0 0 / 6%)', 'rgb(255 255 255 / 10%)'],
    '--color-text-primary': ['#171717', '#f5f5f5'],
    '--color-text-secondary': ['#666666', '#aaaaaa'],
    '--focus-outline-color': 'var(--color-accent)',
    '--radius-chat': '24px',
    '--shadow-low': '0 2px 12px light-dark(#0000000c, #00000040)',
    '--shadow-med': '0 12px 36px light-dark(#00000014, #00000070)',
  },
  adaptations: {
    rules: [
      {
        when: { motion: 'reduce' },
        value: {
          components: {
            button: { 'variant:attachment': { transitionDuration: '0ms' } },
            'chat-composer-drawer': { base: { transitionDuration: '0ms' } },
          },
        },
      },
    ],
  },
  components: {
    markdown: {
      base: {
        ':is([data-tone="thinking"])': {
          '--color-text-primary': 'var(--color-text-secondary)',
        },
      },
    },
    button: {
      'variant:attachment': {
        backgroundColor: 'transparent',
        color: 'var(--color-icon-primary)',
        transitionProperty: 'transform',
        transitionDuration: 'var(--duration-medium)',
        transitionTimingFunction: 'var(--ease-standard)',
        ':is([aria-expanded="true"])': { transform: 'rotate(90deg)' },
      },
    },
    'chat-composer-drawer': {
      base: {
        maxHeight: '32rem',
        transitionProperty: 'transform, opacity',
        transitionDuration: 'var(--duration-medium)',
        transitionTimingFunction: 'var(--ease-standard)',
        ':is([data-menu-open="false"])': {
          maxHeight: '0px',
          paddingBlockStart: '0px',
          paddingBlockEnd: '0px',
          marginBlockEnd: '0px',
          opacity: '0',
          transform: 'translateY(0.5rem)',
          pointerEvents: 'none',
        },
      },
    },
    'app-shell': { base: { backgroundColor: 'var(--color-background-body)' } },
    'side-nav': {
      base: { backgroundColor: 'light-dark(rgb(250 250 250 / 82%), rgb(18 18 18 / 86%))' },
    },
    dialog: {
      base: { backdropFilter: 'blur(24px)', backgroundColor: 'var(--color-background-popover)' },
    },
    'dropdown-menu': { base: { backdropFilter: 'blur(20px)' } },
    popover: { base: { backdropFilter: 'blur(20px)' } },
    'chat-composer': { base: { backdropFilter: 'blur(20px)' } },
  },
});
