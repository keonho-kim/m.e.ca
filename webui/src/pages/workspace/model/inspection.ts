// Purpose: Own one conversation inspector independently of virtualized message lifetimes.
// Pattern: Discriminated view state with stable actions, durable references and bounded fallback metadata.
// Usage: Chat passes actions to history and live renderers and mounts one inspector host.
// Related: webui/src/pages/workspace/ui/inspection/conversation-inspector.tsx.
import { useCallback, useMemo, useState } from 'react';
import type { DetailRequestJson } from '@/shared/api';
export type ToolRequestInspection = {
  name: string;
  reference?: DetailRequestJson['tool'];
  args?: unknown;
};
export type Inspection =
  | { kind: 'original'; source: DetailRequestJson }
  | { kind: 'tool-request'; request: ToolRequestInspection }
  | { kind: 'image'; url: string };
export type InspectionActions = {
  original: (source: DetailRequestJson) => void;
  history: (sequence: string) => void;
  request: (request: ToolRequestInspection) => void;
  image: (url: string) => void;
};
export function useConversationInspection() {
  const [value, setValue] = useState<Inspection | null>(null);
  const actions = useMemo<InspectionActions>(
    () => ({
      original: (source) => setValue({ kind: 'original', source }),
      history: (sequence) => setValue({ kind: 'original', source: { historySequence: sequence } }),
      request: (request) => setValue({ kind: 'tool-request', request }),
      image: (url) => setValue({ kind: 'image', url }),
    }),
    []
  );
  const close = useCallback(() => setValue(null), []);
  return { value, actions, close };
}
