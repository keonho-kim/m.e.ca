// Purpose: Own the browser server cache and transport retry defaults.
// Pattern: TanStack Query client composition.
// Usage: The app provider and native API consumers share this instance.
// Related: webui/src/app/main.tsx, webui/src/shared/lib/errors/index.ts.

import { QueryClient } from '@tanstack/react-query';
import { requestAccessAllowed } from '@/shared/api/access-lifetime';
import { ApiError } from '@/shared/lib/errors';

export const queries = new QueryClient({
  defaultOptions: {
    queries: {
      enabled: requestAccessAllowed,
      staleTime: 15000,
      gcTime: 60000,
      retry: (count, error) =>
        count < 2 && error instanceof ApiError && error.code === 'resource_exhausted',
      retryDelay: (count) => 100 * (count + 1) + Math.floor(Math.random() * 100),
      refetchOnWindowFocus: false,
    },
    mutations: { retry: false },
  },
});
