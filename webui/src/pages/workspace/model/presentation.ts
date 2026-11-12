// Purpose: Reduce hidden chat presentation work without suspending its event connection or task state.
// Pattern: External-store observation of workspace visibility and native modal lifetime.
// Usage: Event batching and live Markdown share the same presentation admission decision.
// Related: webui/src/pages/workspace/api/events.ts, webui/src/pages/workspace/ui/chat/live-output.tsx.
import { useSyncExternalStore } from 'react';
import { useWorkspaceStore } from '@/pages/workspace/model/store';

export function chatPresentationBlocked() {
  const state = useWorkspaceStore.getState();
  return (
    state.mode !== 'chat' ||
    !!state.settings ||
    state.picker ||
    !!state.reviewTarget ||
    (typeof document !== 'undefined' &&
      (document.hidden || !!document.querySelector('dialog[open]')))
  );
}

export function observeChatPresentation(listener: () => void) {
  let previous = chatPresentationBlocked();
  const update = () => {
    const current = chatPresentationBlocked();
    if (current !== previous) {
      previous = current;
      listener();
    }
  };
  const release = useWorkspaceStore.subscribe((state, before) => {
    if (
      state.mode !== before.mode ||
      state.settings !== before.settings ||
      state.picker !== before.picker ||
      state.reviewTarget !== before.reviewTarget
    )
      update();
  });
  const toggle = (event: Event) => {
    if (event.target instanceof HTMLDialogElement) update();
  };
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', update);
    document.addEventListener('toggle', toggle, true);
  }
  return () => {
    release();
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', update);
      document.removeEventListener('toggle', toggle, true);
    }
  };
}

export function useChatPresentationBlocked() {
  return useSyncExternalStore(observeChatPresentation, chatPresentationBlocked, () => false);
}
