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
  motion: { fast: 130, medium: 220, slow: 360, ratio: 0.8, character: 'crisp' },
  localTokens: {
    '--meca-glass-surface': ['rgb(255 255 255 / 78%)', 'rgb(23 25 31 / 84%)'],
    '--meca-glass-filter': 'blur(18px) saturate(115%)',
    '--meca-glass-edge': [
      'inset 0 1px 0 rgb(255 255 255 / 94%), inset 0 0 0 1px rgb(20 27 40 / 8%)',
      'inset 0 1px 0 rgb(255 255 255 / 11%), inset 0 0 0 1px rgb(255 255 255 / 6%)',
    ],
    '--meca-glass-shadow': [
      '0 12px 28px -16px rgb(20 27 40 / 20%)',
      '0 16px 40px -20px rgb(0 0 0 / 65%)',
    ],
    '--meca-thinking-surface': ['rgb(235 238 242 / 70%)', 'rgb(31 34 41 / 84%)'],
    '--meca-thinking-ink': ['#606774', '#b4bac5'],
  },
  tokens: {
    '--font-family-body': font,
    '--font-family-heading': font,
    '--color-background-body': ['#f2f4f7', '#0d0f13'],
    '--color-background-surface': ['#fcfdff', '#121419'],
    '--color-background-card': ['#ffffff', '#1a1d23'],
    '--color-background-popover': ['rgb(255 255 255 / 88%)', 'rgb(26 29 35 / 92%)'],
    '--color-background-muted': ['#edf0f4', '#1e2128'],
    '--color-accent': ['#191d24', '#eef0f5'],
    '--color-on-accent': ['#ffffff', '#191d24'],
    '--color-accent-muted': ['rgb(0 0 0 / 6%)', 'rgb(255 255 255 / 10%)'],
    '--color-text-primary': ['#191d24', '#eef0f5'],
    '--color-text-secondary': ['#616876', '#adb3bf'],
    '--focus-outline-color': 'light-dark(#285ba2, #b6c7e2)',
    '--radius-chat': '24px',
    '--shadow-low': '0 2px 12px light-dark(#0000000c, #00000040)',
    '--shadow-med': '0 12px 36px light-dark(#00000014, #00000070)',
  },
  adaptations: {
    rules: [
      {
        when: { pointer: 'coarse' },
        value: {
          components: {
            'collapsible-trigger': { base: { minHeight: 'var(--spacing-12)' } },
          },
        },
      },
      {
        when: { motion: 'reduce' },
        value: {
          components: {
            button: { 'variant:attachment': { transitionDuration: '0ms' } },
            dialog: { base: { animationName: 'none' } },
          },
        },
      },
    ],
  },
  components: {
    center: {
      base: {
        ':is(.startup-screen)': {
          backgroundImage:
            'linear-gradient(145deg, var(--color-background-surface), var(--color-background-body))',
        },
      },
    },
    stack: {
      base: {
        ':is(.composer-send-hint)': {
          '--color-text-primary': 'currentColor',
          '--color-neutral': 'color-mix(in srgb, currentColor 12%, transparent)',
          '--color-border-emphasized': 'color-mix(in srgb, currentColor 30%, transparent)',
        },
        ':is(.startup-surface)': {
          backgroundColor: 'var(--meca-glass-surface)',
          WebkitBackdropFilter: 'var(--meca-glass-filter)',
          backdropFilter: 'var(--meca-glass-filter)',
          borderRadius: 'var(--radius-container)',
          boxShadow: 'var(--meca-glass-edge), var(--meca-glass-shadow)',
        },
      },
    },
    'collapsible-trigger': { base: { minHeight: 'var(--size-element-md)' } },
    collapsible: {
      base: {
        ':is([data-tone="tool"], [data-tone="request"])': {
          backgroundColor: 'transparent',
          borderWidth: '0',
          boxShadow: 'none',
          padding: '0',
          width: '100%',
          minWidth: '0',
        },
        ':is([data-tone="thinking"])': {
          backgroundColor: 'var(--meca-thinking-surface)',
          borderRadius: 'var(--radius-container)',
          padding: 'var(--spacing-4)',
          '--color-text-primary': 'var(--meca-thinking-ink)',
          '--color-text-secondary': 'var(--meca-thinking-ink)',
        },
      },
    },
    'scrollable-area': {
      base: {
        ':is(.tool-request-content)': {
          maxHeight: 'min(40dvh, calc(var(--spacing-8) * 10))',
          backgroundColor: 'var(--color-background-surface)',
        },
      },
    },
    markdown: {
      base: {
        fontSize: 'var(--text-body-size)',
        lineHeight: '1.7',
        ':is([data-tone="thinking"])': {
          '--color-text-primary': 'var(--color-text-secondary)',
        },
      },
    },
    button: {
      'variant:image-remove': {
        position: 'absolute',
        top: 'var(--spacing-1)',
        right: 'var(--spacing-1)',
        height: 'var(--spacing-6)',
        width: 'var(--spacing-6)',
        minWidth: 'var(--spacing-6)',
        borderRadius: 'var(--radius-full)',
        backgroundColor: 'var(--color-overlay)',
        color: 'var(--color-on-dark)',
      },
      'variant:attachment': {
        backgroundColor: 'transparent',
        color: 'var(--color-icon-primary)',
        transitionProperty: 'transform',
        transitionDuration: 'var(--duration-medium)',
        transitionTimingFunction: 'var(--ease-standard)',
        ':is([aria-expanded="true"])': { transform: 'rotate(90deg)' },
      },
    },
    'app-shell': { base: { backgroundColor: 'var(--color-background-body)' } },
    'side-nav': {
      base: {
        backgroundColor: 'var(--meca-glass-surface)',
        WebkitBackdropFilter: 'var(--meca-glass-filter)',
        backdropFilter: 'var(--meca-glass-filter)',
        boxShadow: 'var(--meca-glass-edge)',
      },
    },
    dialog: {
      base: {
        animationName: 'meca-dialog-enter',
        animationDuration: 'var(--duration-medium)',
        animationTimingFunction: 'var(--ease-entrance)',
        backgroundColor: 'var(--color-background-card)',
        WebkitBackdropFilter: 'none',
        backdropFilter: 'none',
        boxShadow: 'var(--shadow-high)',
      },
    },
    drawer: {
      base: {
        backgroundColor: 'var(--color-background-card)',
        WebkitBackdropFilter: 'none',
        backdropFilter: 'none',
      },
    },
    'alert-dialog': {
      base: {
        backgroundColor: 'var(--color-background-card)',
        WebkitBackdropFilter: 'none',
        backdropFilter: 'none',
      },
    },
    lightbox: {
      base: {
        backgroundColor: 'var(--color-on-light)',
        backdropFilter: 'none',
        '::backdrop': { backdropFilter: 'none' },
      },
    },
    'dropdown-menu': {
      base: {
        WebkitBackdropFilter: 'var(--meca-glass-filter)',
        backdropFilter: 'var(--meca-glass-filter)',
        boxShadow: 'var(--meca-glass-edge), var(--shadow-med)',
      },
    },
    popover: {
      base: {
        WebkitBackdropFilter: 'var(--meca-glass-filter)',
        backdropFilter: 'var(--meca-glass-filter)',
        boxShadow: 'var(--meca-glass-edge), var(--shadow-med)',
      },
    },
    'chat-composer': {
      base: {
        borderRadius: 'var(--radius-chat)',
        WebkitBackdropFilter: 'var(--meca-glass-filter)',
        backdropFilter: 'var(--meca-glass-filter)',
        backgroundColor: 'var(--meca-glass-surface)',
        boxShadow: 'var(--meca-glass-edge), var(--meca-glass-shadow)',
      },
    },
    toolbar: {
      base: {
        ':is(.workspace-header)': {
          backgroundColor: 'var(--meca-glass-surface)',
          WebkitBackdropFilter: 'var(--meca-glass-filter)',
          backdropFilter: 'var(--meca-glass-filter)',
          boxShadow: 'var(--meca-glass-edge)',
        },
      },
    },
    'chat-tool-calls': {
      base: {
        width: '100%',
        minWidth: '0',
        marginBlockStart: '0',
        backgroundColor: 'transparent',
        '--color-overlay-hover': 'transparent',
        textDecorationLine: 'underline',
        textDecorationColor: 'var(--color-border-emphasized)',
        textUnderlineOffset: '0.25em',
        textDecorationThickness: 'var(--border-thin)',
      },
    },
    carousel: {
      base: {
        paddingBlock: 'var(--spacing-2)',
        backgroundColor: 'var(--color-background-popover)',
        borderRadius: 'var(--radius-container)',
        boxShadow: 'var(--shadow-low)',
        WebkitBackdropFilter: 'var(--meca-glass-filter)',
        backdropFilter: 'var(--meca-glass-filter)',
      },
    },
  },
});
