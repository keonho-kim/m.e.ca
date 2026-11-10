// Purpose: Supply cancellable native command and skill metadata to Astryx input triggers.
// Pattern: SearchSource adapters at the API boundary.
// Usage: Composer supplies these trigger definitions to ChatComposerInput.
// Related: webui/src/shared/api/client.ts, webui/src/pages/workspace/ui/composer/composer.tsx.

import type { ChatComposerTrigger } from '@astryxdesign/core/Chat';
import {
  createStaticSource,
  type SearchableItem,
  type SearchSource,
  TypeaheadItem,
} from '@astryxdesign/core/Typeahead';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useMemo } from 'react';
import { reportError } from '@/entities/server-session';
import { call } from '@/shared/api';
export function useComposerTriggers(workspace: string): ChatComposerTrigger[] {
  const catalog = useQuery({
    queryKey: ['command-catalog'],
    queryFn: async ({ signal }) =>
      (await call('', { commandCatalog: {} }, signal)).commandCatalog?.items ?? [],
  });
  const commands = useMemo(
    () =>
      createStaticSource(
        (catalog.data ?? []).map((item) => ({
          id: item.name ?? '',
          label: item.name ?? '',
          auxiliaryData: item.description,
        }))
      ),
    [catalog.data]
  );
  const skills = useMemo<SearchSource>(() => {
    let controller: AbortController | null = null;
    const search = async (prefix: string) => {
      controller?.abort();
      const owned = new AbortController();
      controller = owned;
      try {
        const reply = await call(workspace, { suggestSkills: { prefix } }, owned.signal);
        return (reply.skills?.items ?? []).slice(0, 8).map((skill) => ({
          id: skill.id || skill.name || '',
          label: skill.name ?? '',
          auxiliaryData: skill.description,
        }));
      } catch (error) {
        if (!owned.signal.aborted) reportError(error);
        return [];
      }
    };
    return { search, bootstrap: () => search(''), cancel: () => controller?.abort() };
  }, [workspace]);
  useEffect(() => () => skills.cancel?.(), [skills]);
  return useMemo(() => {
    const renderItem = (item: SearchableItem) => (
      <TypeaheadItem
        item={item}
        description={typeof item.auxiliaryData === 'string' ? item.auxiliaryData : undefined}
      />
    );
    return [
      {
        character: '!',
        searchSource: commands,
        renderItem,
        onSelect: (item) => (item.label.startsWith('!') ? item.label : `!${item.label}`),
      },
      {
        character: '#',
        searchSource: skills,
        renderItem,
        onSelect: (item) => ({ value: `#${item.id}`, label: `#${item.label}`, variant: 'neutral' }),
      },
    ];
  }, [commands, skills]);
}
