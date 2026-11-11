// Purpose: Verify conversation deletion releases token totals and bodies belonging to that conversation only.
// Pattern: Local cache ownership regression test.
// Usage: bun test src/pages/workspace/model/deletion.test.ts.
// Related: webui/src/pages/workspace/model/deletion.ts.
import { expect, test } from 'bun:test';
import { forgetConversation } from '@/pages/workspace/model/deletion';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { queries } from '@/shared/api';

test('deletion removes usage and history without removing another session totals', () => {
  const original = useWorkspaceStore.getState();
  try {
    queries.clear();
    for (const key of [
      ['session-usage', 'w', 'one'],
      ['history', 'w', 'one', '0'],
      ['session-usage', 'w', 'two'],
    ])
      queries.setQueryData(key, {});
    forgetConversation({ workspace: 'w', session: 'one' });
    expect(queries.getQueryData(['session-usage', 'w', 'one'])).toBeUndefined();
    expect(queries.getQueryData(['history', 'w', 'one', '0'])).toBeUndefined();
    expect(queries.getQueryData(['session-usage', 'w', 'two'])).toEqual({});
  } finally {
    queries.clear();
    useWorkspaceStore.setState(original);
  }
});
