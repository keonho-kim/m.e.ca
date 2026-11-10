// Purpose: Wire React, Astryx, routing and query ownership for the embedded application.
// Pattern: Browser composition root.
// Usage: The static HTML entry imports this module.
// Related: webui/src/app/routes/application.tsx, webui/src/app/styles/theme.ts, webui/vite.config.ts.

import { Button } from '@astryxdesign/core/Button';
import { Center } from '@astryxdesign/core/Center';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { InternationalizationProvider } from '@astryxdesign/core/i18n';
import korean from '@astryxdesign/core/locales/ko-KR.generated.js';
import { Theme } from '@astryxdesign/core/theme';
import { QueryClientProvider } from '@tanstack/react-query';
import { createRootRoute, createRoute, createRouter, RouterProvider } from '@tanstack/react-router';
import { StrictMode, useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from '@/app/routes/application';
import { mecaTheme } from '@/app/styles/generated/meca';
import { reportError, useServerSession } from '@/entities/server-session';
import { queries } from '@/shared/api';
import { useAppearance, useResolvedTheme } from '@/shared/config';
import { languageCode, useLocale } from '@/shared/i18n';
import '@/app/styles/base.css';
import '@/app/styles/generated/meca.css';

const coreMessages = { 'ko-KR': korean };

function Recover({ reset }: { reset: () => void }) {
  const t = useLocale();
  return (
    <Center height="100dvh" padding={6}>
      <EmptyState
        title="MECA"
        description={t('readError')}
        actions={
          <Button
            label={t('retry')}
            onClick={() => {
              void import('@/pages/workspace')
                .then((page) => {
                  page.resetWorkspaceView();
                  reset();
                })
                .catch(reportError);
            }}
          />
        }
      />
    </Center>
  );
}
const rootRoute = createRootRoute({ component: App, errorComponent: Recover });
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
