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
  // Native close/toggle events can run after React has detached the dialog.
  // Watch only its ancestor child lists so removal is observed without watching
  // every streamed text node in the document.
  let removal: MutationObserver | undefined;
  const watchRemoval = () => {
    removal?.disconnect();
    if (!removal || typeof document === 'undefined') return;
    const parents = new Set<Element>();
    for (const dialog of document.querySelectorAll('dialog[open]'))
      for (let parent = dialog.parentElement; parent; parent = parent.parentElement)
        parents.add(parent);
    for (const parent of parents) removal.observe(parent, { childList: true });
  };
  const update = () => {
    watchRemoval();
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
  let fallback: MutationObserver | undefined;
  if (typeof document !== 'undefined') {
    removal = new MutationObserver(update);
    watchRemoval();
    document.addEventListener('visibilitychange', update);
    document.addEventListener('toggle', toggle, true);
    if (typeof HTMLDialogElement !== 'undefined' && !('ontoggle' in HTMLDialogElement.prototype)) {
      fallback = new MutationObserver(update);
      fallback.observe(document.body, {
        subtree: true,
        attributes: true,
        attributeFilter: ['open'],
      });
    }
  }
  return () => {
    removal?.disconnect();
    fallback?.disconnect();
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
