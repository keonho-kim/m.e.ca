// Purpose: Accept bounded composer images from file inputs, drops and clipboard shortcuts.
// Pattern: Browser input adapter with conversation-scoped clipboard lifetime.
// Usage: Composer shares these handlers with Astryx input and drawer components.
// Related: webui/src/pages/workspace/ui/composer/composer.tsx, webui/src/pages/workspace/model/store.ts, webui/src/shared/lib/errors/index.ts.

import { useHotkeys } from '@astryxdesign/core/hooks';
import type { ClipboardEvent, DragEvent, FocusEvent } from 'react';
import { useEffect, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import type { CapabilitiesJson } from '@/shared/api';
import { failure } from '@/shared/lib/errors';

export function useComposerImages(target: Target, caps?: CapabilitiesJson) {
  const pending = useRef<AbortController | null>(null);
  const [focused, setFocused] = useState(false);
  useEffect(() => {
    if (!target.workspace || !target.session) return;
    return () => {
      pending.current?.abort();
      pending.current = null;
    };
  }, [target.workspace, target.session]);
  const countLimit = caps?.inputImages ?? 8,
    fileLimit = Number(caps?.imageBytes ?? 67108864),
    totalLimit = Number(caps?.inputImageBytes ?? 134217728);
  const attach = (files: File[]) => {
    if (!target.session || !files.length) return;
    const state = useWorkspaceStore.getState();
    if (files.some((file) => !file.type.startsWith('image/'))) {
      reportError(failure('image_type'));
      return;
    }
    const images = [...(state.inputs[targetKey(target)]?.files ?? []), ...files];
    if (
      images.length > countLimit ||
      images.some((file) => file.size > fileLimit) ||
      images.reduce((size, file) => size + file.size, 0) > totalLimit
    ) {
      reportError(failure('image_limit'));
      return;
    }
    state.patchInput(target, { files: images });
  };
  const readClipboard = async () => {
    if (pending.current || !target.session) return;
    const controller = new AbortController();
    const isAborted = () => controller.signal.aborted;
    pending.current = controller;
    try {
      const clipboard = navigator.clipboard as Partial<Pick<Clipboard, 'read'>> | undefined;
      if (!clipboard?.read) throw failure('clipboard_unavailable');
      const items = await clipboard.read();
      if (isAborted()) return;
      const images = items.flatMap((item) => {
        const type = item.types.find((type) => type.startsWith('image/'));
        return type ? [{ item, type }] : [];
      });
      if (!images.length) throw failure('clipboard_empty');
      const existing = useWorkspaceStore.getState().inputs[targetKey(target)]?.files ?? [];
      if (existing.length + images.length > countLimit) throw failure('image_limit');
      let total = existing.reduce((size, file) => size + file.size, 0);
      const files: File[] = [];
      for (const { item, type } of images) {
        const blob = await item.getType(type);
        if (isAborted()) return;
        total += blob.size;
        if (blob.size > fileLimit || total > totalLimit) throw failure('image_limit');
        const extension = type.slice('image/'.length).replace(/[^a-z0-9]/gi, '') || 'png';
        files.push(new File([blob], `clipboard-${files.length + 1}.${extension}`, { type }));
      }
      attach(files);
    } catch (error) {
      if (!isAborted())
        reportError(error instanceof DOMException ? failure('clipboard_unavailable') : error);
    } finally {
      if (pending.current === controller) pending.current = null;
    }
  };
  const disabled = !focused || !target.session;
  useHotkeys([
    {
      keys: 'alt+v',
      allowInInputs: true,
      isDisabled: disabled,
      onPress: () => void readClipboard(),
    },
    {
      keys: 'alt+√',
      allowInInputs: true,
      isDisabled: disabled,
      onPress: () => void readClipboard(),
    },
    {
      keys: 'ctrl+v',
      allowInInputs: true,
      isDisabled: disabled || !/mac/i.test(navigator.platform),
      onPress: () => void readClipboard(),
    },
  ]);
  return {
    attach,
    onFocusCapture: () => setFocused(true),
    onBlurCapture: (event: FocusEvent<HTMLDivElement>) => {
      if (!event.currentTarget.contains(event.relatedTarget)) setFocused(false);
    },
    onDragOver: (event: DragEvent) => {
      if (event.dataTransfer.types.includes('Files')) event.preventDefault();
    },
    onDrop: (event: DragEvent) => {
      if (event.defaultPrevented || !event.dataTransfer.files.length) return;
      event.preventDefault();
      attach(Array.from(event.dataTransfer.files));
    },
    onPasteCapture: (event: ClipboardEvent) => {
      if (event.defaultPrevented || !event.clipboardData.files.length) return;
      event.preventDefault();
      event.stopPropagation();
      attach(Array.from(event.clipboardData.files));
    },
  };
}
