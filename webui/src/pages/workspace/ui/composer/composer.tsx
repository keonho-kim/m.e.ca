// Purpose: Compose commands, steering and immutable follow-ups with bounded image input.
// Pattern: Input controller over shared native actions.
// Usage: The workspace keeps this composer available in chat, files and skill views.
// Related: webui/src/pages/workspace/ui/composer/composer-actions.tsx, webui/src/pages/workspace/model/execution.ts, webui/src/pages/workspace/model/file-references.ts.

import { Center } from '@astryxdesign/core/Center';
import {
  ChatComposer,
  ChatComposerInput,
  type ChatComposerInputHandle,
  ChatSendButton,
} from '@astryxdesign/core/Chat';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Token } from '@astryxdesign/core/Token';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useId, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { stateOptions } from '@/pages/workspace/api/queries';
import { command, settings } from '@/pages/workspace/model/commands';
import { send } from '@/pages/workspace/model/execution';
import { selectReference, waitSelections } from '@/pages/workspace/model/file-references';
import {
  emptyInput,
  type Target,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { checkSubmissionLimits } from '@/pages/workspace/model/submission-limits';
import { activeTask } from '@/pages/workspace/model/task-state';
import { useComposerImages } from '@/pages/workspace/model/use-composer-images';
import { useComposerTriggers } from '@/pages/workspace/model/use-composer-triggers';
import { AttachmentPanel } from '@/pages/workspace/ui/composer/attachment-panel';
import {
  ComposerActions,
  ComposerSendActions,
} from '@/pages/workspace/ui/composer/composer-actions';
import { Queue } from '@/pages/workspace/ui/composer/queue';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { failure } from '@/shared/lib/errors';

const fileKeys = new WeakMap<File, string>();
function fileKey(file: File) {
  let key = fileKeys.get(file);
  if (!key) {
    key = crypto.randomUUID();
    fileKeys.set(file, key);
  }
  return key;
}

export function Composer({ target }: { target: Target }) {
  const t = useLocale(),
    input = useWorkspaceStore((s) => s.inputs[targetKey(target)]) ?? emptyInput();
  const task = useWorkspaceStore((state) => activeTask(target, state.tasks));
  const editor = useRef<ChatComposerInputHandle>(null);
  const triggers = useComposerTriggers(target.workspace);
  const [expanded, setExpanded] = useState(false);
  const panelId = useId();
  const [busy, setBusy] = useState(false);
  const [inputEpoch, setInputEpoch] = useState(0);
  const state = useQuery({ ...stateOptions(target), enabled: !!target.session });
  const capabilities = useQuery({
    queryKey: ['capabilities'],
    queryFn: async ({ signal }) => (await call('', { capabilities: {} }, signal)).capabilities!,
  });
  useEffect(() => {
    const controller = new AbortController();
    if (target.session)
      void waitSelections(target)
        .then(() =>
          call(
            target.workspace,
            { fileSelections: { sessionId: target.session } },
            controller.signal
          )
        )
        .then((reply) => {
          if (!controller.signal.aborted)
            useWorkspaceStore
              .getState()
              .patchInput(target, { paths: reply.fileSelections?.paths ?? [] });
        })
        .catch((error) => {
          if (!controller.signal.aborted) reportError(error);
        });
    return () => controller.abort();
  }, [target]);
  const patch = (text: string) => {
    if (!useWorkspaceStore.getState().patchInput(target, { text }))
      setInputEpoch((value) => value + 1);
  };
  const submit = async (reserve = false) => {
    if (busy || !target.session) return;
    setBusy(true);
    try {
      if (input.text.trimStart().startsWith('!')) {
        if (await command(target, input.text)) patch('');
      } else {
        if (!state.data?.model?.model) {
          settings('model', target);
          return;
        }
        await waitSelections(target);
        const captured = useWorkspaceStore.getState().inputs[targetKey(target)] ?? input;
        if (!capabilities.data) throw failure('invalid_response');
        checkSubmissionLimits(captured, capabilities.data);
        await send(target, captured, reserve);
      }
    } catch (error) {
      reportError(error);
    } finally {
      setBusy(false);
    }
  };
  const images = useComposerImages(target, capabilities.data);
  return (
    <Center width="100%">
      <VStack width="100%" maxWidth={760} padding={4} gap={3}>
        <Queue target={target} />
        <ChatComposer
          value={input.text}
          onChange={patch}
          onSubmit={() => {
            void submit();
          }}
          isDisabled={!target.session}
          className="composer"
          onDragOver={images.onDragOver}
          onDrop={images.onDrop}
          onPasteCapture={images.onPasteCapture}
          onFocusCapture={images.onFocusCapture}
          onBlurCapture={images.onBlurCapture}
          onKeyDown={(event) => {
            if (expanded && event.key === 'Escape' && !event.defaultPrevented) {
              event.preventDefault();
              setExpanded(false);
              editor.current?.focus();
            }
          }}
          input={
            <ChatComposerInput
              key={inputEpoch}
              handleRef={editor}
              label={t('input')}
              placeholder={t('placeholder')}
              triggers={triggers}
              hasHistory={false}
              onFiles={images.attach}
              onKeyDown={(event) => {
                if (event.nativeEvent.isComposing || event.nativeEvent.keyCode === 229) return;
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault();
                  void submit(event.ctrlKey || event.metaKey);
                }
              }}
            />
          }
          drawer={
            <AttachmentPanel
              id={panelId}
              expanded={expanded}
              onExpandedChange={(value) => {
                setExpanded(value);
                if (!value) editor.current?.focus();
              }}
              attach={images.attach}
              maxFiles={capabilities.data?.inputImages ?? 8}
              maxSize={Number(capabilities.data?.imageBytes ?? 67108864)}
            />
          }
          headerActions={
            input.paths.length || input.files.length ? (
              <HStack gap={2} wrap="wrap">
                {input.paths.map((path) => (
                  <Token
                    key={path}
                    label={path}
                    onRemove={() => {
                      void selectReference(target, path, false).catch(reportError);
                    }}
                  />
                ))}
                {input.files.map((file, index) => (
                  <Token
                    key={fileKey(file)}
                    label={file.name}
                    onRemove={() =>
                      useWorkspaceStore
                        .getState()
                        .patchInput(target, { files: input.files.filter((_, i) => index !== i) })
                    }
                  />
                ))}
              </HStack>
            ) : undefined
          }
          footerActions={
            <ComposerActions
              target={target}
              teamEnabled={!!state.data?.team}
              busy={busy}
              expanded={expanded}
              onExpandedChange={setExpanded}
              panelId={panelId}
            />
          }
          sendActions={
            <HStack gap={1}>
              <ComposerSendActions target={target} task={task?.id} busy={busy} submit={submit} />
            </HStack>
          }
          sendButton={
            <ChatSendButton
              aria-label={task ? t('steer') : t('send')}
              className="send-button"
              isDisabled={busy || !target.session || (!input.text.trim() && !input.files.length)}
              onSend={() => {
                void submit();
              }}
            />
          }
        />
      </VStack>
    </Center>
  );
}
