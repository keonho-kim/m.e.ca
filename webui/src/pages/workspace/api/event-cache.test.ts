// Purpose: Verify conversation lifecycle facts refresh only the owning project's metadata.
// Pattern: Query projection boundary regression test.
// Usage: bun test src/pages/workspace/api/event-cache.test.ts.
// Related: webui/src/pages/workspace/api/event-cache.ts.
import { expect, test } from 'bun:test';
import { projectEventCache } from '@/pages/workspace/api/event-cache';
import { queries } from '@/shared/api';

test('remote conversation creation and closure refresh project listings without invalidating unrelated bodies', () => {
  try {
    for (const kind of ['session_opened', 'session_closed']) {
      queries.clear();
      const sessions = ['sessions', 'alpha'];
      const other = ['sessions', 'beta'];
      const history = ['history', 'beta', 'one', '0'];
      for (const key of [sessions, other, history, ['workspaces']]) queries.setQueryData(key, {});
      projectEventCache([{ kind, workspaceId: 'alpha', sessionId: 'new' }]);
      expect(queries.getQueryState(sessions)?.isInvalidated).toBe(true);
      expect(queries.getQueryState(['workspaces'])?.isInvalidated).toBe(true);
      expect(queries.getQueryState(other)?.isInvalidated).toBe(false);
      expect(queries.getQueryState(history)?.isInvalidated).toBe(false);
    }
  } finally {
    queries.clear();
  }
});

test('model changes and failed commits refresh their own state and credential availability', () => {
  try {
    for (const kind of ['model_changed', 'model_unavailable']) {
      queries.clear();
      const state = ['state', 'alpha', 'one'],
        other = ['state', 'beta', 'two'];
      for (const key of [state, other, ['capabilities']]) queries.setQueryData(key, {});
      projectEventCache([{ kind, workspaceId: 'alpha', sessionId: 'one' }]);
      expect(queries.getQueryState(state)?.isInvalidated).toBe(true);
      expect(queries.getQueryState(['capabilities'])?.isInvalidated).toBe(true);
      expect(queries.getQueryState(other)?.isInvalidated).toBe(false);
    }
  } finally {
    queries.clear();
  }
});
