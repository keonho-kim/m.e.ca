// Purpose: Own the attachment viewer independently of virtualized message lifetimes.
// Pattern: Conversation-owned attachment state with stable actions.
// Usage: Chat passes actions to history and live renderers and mounts one inspector host.
// Related: webui/src/pages/workspace/ui/inspection/conversation-inspector.tsx.
import { useCallback, useMemo, useState } from 'react';
export type Inspection = { kind: 'image'; url: string };
export type InspectionActions = {
  image: (url: string) => void;
};
export function useConversationInspection() {
  const [value, setValue] = useState<Inspection | null>(null);
  const actions = useMemo<InspectionActions>(
    () => ({
      image: (url) => setValue({ kind: 'image', url }),
    }),
    []
  );
  const close = useCallback(() => setValue(null), []);
  return { value, actions, close };
}
