// Purpose: Wire React, Astryx, routing and query ownership for the embedded application.
// Pattern: Browser composition root.
// Usage: The static HTML entry imports this module.
// Related: webui/src/app/routes/application.tsx, webui/src/app/styles/theme.ts, webui/vite.config.ts.

import { InternationalizationProvider } from '@astryxdesign/core/i18n';
import korean from '@astryxdesign/core/locales/ko-KR.generated.js';
import { Theme } from '@astryxdesign/core/theme';
import { QueryClientProvider } from '@tanstack/react-query';
import { createRootRoute, createRoute, createRouter, RouterProvider } from '@tanstack/react-router';
import { StrictMode, useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from '@/app/routes/application';
import { mecaTheme } from '@/app/styles/generated/meca';
import { useServerSession } from '@/entities/server-session';
import { queries } from '@/shared/api';
import { useAppearance, useResolvedTheme } from '@/shared/config';
import { languageCode } from '@/shared/i18n';
import { watchReload } from '@/shared/lib/reload';
import { Recovery } from '@/shared/ui/recovery';
import '@/app/styles/base.css';
import '@/app/styles/generated/meca.css';

const coreMessages = { 'ko-KR': korean };

const rootRoute = createRootRoute({ component: App, errorComponent: Recovery });
const home = createRoute({ getParentRoute: () => rootRoute, path: '/', component: () => null });
const workspace = createRoute({
  getParentRoute: () => rootRoute,
  path: '/w/$workspaceId',
  component: () => null,
});
const conversation = createRoute({
  getParentRoute: () => rootRoute,
  path: '/w/$workspaceId/c/$sessionId',
  component: () => null,
});
const router = createRouter({
  routeTree: rootRoute.addChildren([home, workspace, conversation]),
  basepath: '/ui',
  defaultPreload: false,
  defaultPendingMinMs: 0,
});
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

function Root() {
  useEffect(watchReload, []);
  const mode = useAppearance((s) => s.theme),
    language = useServerSession((s) => s.language);
  const resolvedTheme = useResolvedTheme(mode);
  useEffect(() => {
    document.documentElement.lang = languageCode(language);
  }, [language]);
  return (
    <InternationalizationProvider
      locale={languageCode(language) === 'ko' ? 'ko-KR' : 'en'}
      messages={coreMessages}
    >
      <Theme theme={mecaTheme} mode={resolvedTheme}>
        <QueryClientProvider client={queries}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </Theme>
    </InternationalizationProvider>
  );
}
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Root />
  </StrictMode>
);
