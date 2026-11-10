// Purpose: Own browser-only appearance preferences and resolve the OS scheme through Astryx.
// Pattern: Zustand preference state with the official media-query hook.
// Usage: App theme and theme-selection controls share this owner.
// Related: webui/src/pages/workspace/ui/settings/theme-controls.tsx.

import { useMediaQuery } from '@astryxdesign/core/hooks';
import { create } from 'zustand';
export type Appearance = 'light' | 'dark' | 'system';
function preferredTheme(): Appearance {
  try {
    const value = localStorage.getItem('meca.theme');
    return value === 'dark' || value === 'system' ? value : 'light';
  } catch {
    return 'light';
  }
}
export const useAppearance = create<{ theme: Appearance; setTheme: (theme: Appearance) => void }>(
  (set) => ({
    theme: preferredTheme(),
    setTheme: (theme) => {
      try {
        localStorage.setItem('meca.theme', theme);
      } catch {
        /* Preferences are optional. */
      }
      set({ theme });
    },
  })
);
export function useResolvedTheme(preference: Appearance): 'light' | 'dark' {
  const dark = useMediaQuery('(prefers-color-scheme: dark)');
  return preference === 'system' ? (dark ? 'dark' : 'light') : preference;
}
