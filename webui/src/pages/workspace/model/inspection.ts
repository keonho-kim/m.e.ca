// Purpose: Own one conversation inspector independently of virtualized message lifetimes.
// Pattern: Discriminated view state with stable actions and durable history references.
// Usage: Chat passes actions to history and live renderers and mounts one inspector host.
// Related: webui/src/pages/workspace/ui/inspection/conversation-inspector.tsx.
import { useCallback, useMemo, useState } from 'react';
import type { DetailRequestJson } from '@/shared/api';
export type Inspection =
  | { kind: 'original'; source: DetailRequestJson }
  | { kind: 'image'; url: string };
export type InspectionActions = {
  history: (sequence: string) => void;
  image: (url: string) => void;
};
export function useConversationInspection() {
  const [value, setValue] = useState<Inspection | null>(null);
  const actions = useMemo<InspectionActions>(
    () => ({
      history: (sequence) => setValue({ kind: 'original', source: { historySequence: sequence } }),
      image: (url) => setValue({ kind: 'image', url }),
    }),
    []
  );
  const close = useCallback(() => setValue(null), []);
  return { value, actions, close };
}
