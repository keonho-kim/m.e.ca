// Purpose: Compose commands, steering and immutable follow-ups with bounded image input.
// Pattern: Input controller over shared native actions.
// Usage: The workspace keeps this composer available in chat, files and skill views.
// Related: webui/src/pages/workspace/ui/composer/composer-actions.tsx, webui/src/pages/workspace/model/execution.ts, webui/src/pages/workspace/model/file-references.ts.

import {
  ChatComposer,
  ChatComposerInput,
  type ChatComposerInputHandle,
  ChatSendButton,
} from '@astryxdesign/core/Chat';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { Token } from '@astryxdesign/core/Token';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useId, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { capabilitiesOptions, stateOptions } from '@/pages/workspace/api/queries';
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
import { chatWidth } from '@/pages/workspace/ui/chat/layout';
import { AttachmentMenu } from '@/pages/workspace/ui/composer/attachment-menu';
import {
  ComposerActions,
  ComposerQueueAction,
} from '@/pages/workspace/ui/composer/composer-actions';
import { ComposerImages } from '@/pages/workspace/ui/composer/composer-images';
import { ExecutionStatus } from '@/pages/workspace/ui/composer/execution-status';
import { Queue } from '@/pages/workspace/ui/composer/queue';
import { SkillPicker } from '@/pages/workspace/ui/skills/skill-picker';
import { call, control } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { failure } from '@/shared/lib/errors';

export function Composer({ target, showStatus }: { target: Target; showStatus: boolean }) {
  const t = useLocale(),
    input = useWorkspaceStore((s) => s.inputs[targetKey(target)]) ?? emptyInput();
  const task = useWorkspaceStore((state) => activeTask(target, state.tasks));
  const hasInput = !!(input.text.trim() || input.files.length);
  const showStop = !!task && !hasInput;
  const queueShortcut = /Mac|iPad|iPhone/.test(navigator.userAgent) ? '⌘↵' : 'Ctrl+↵';
  const editor = useRef<ChatComposerInputHandle>(null);
  const focusInput = useCallback(() => editor.current?.focus(), []);
  const sendLabel = useId();
  const triggers = useComposerTriggers(target.workspace);
  const [skillsOpen, setSkillsOpen] = useState(false);
  const previousSkillsOpen = useRef(false);
  useEffect(() => {
    if (previousSkillsOpen.current && !skillsOpen) editor.current?.focus();
    previousSkillsOpen.current = skillsOpen;
  }, [skillsOpen]);
  const [busy, setBusy] = useState(false);
  const [inputEpoch, setInputEpoch] = useState(0);
  const state = useQuery({ ...stateOptions(target), enabled: !!target.session });
  const capabilities = useQuery(capabilitiesOptions);
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
    if (busy || !target.session || !hasInput) return;
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
    <VStack width={chatWidth} padding={4} gap={3} className="mx-auto">
      <Text id={sendLabel} hidden>
        {t(showStop ? 'stop' : task ? 'steer' : 'send')}
      </Text>
      <Queue target={target} />
      <ComposerImages target={target} onEmpty={focusInput} />
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
        input={
          // Keep the empty editable stretched when a modal restores background interactivity.
          <ChatComposerInput
            className="grid"
            key={inputEpoch}
            handleRef={editor}
            label={t('input')}
            placeholder={t('placeholder')}
            triggers={triggers}
            hasHistory={false}
            onFiles={images.attach}
            onKeyDown={(event) => {
              if (
                event.defaultPrevented ||
                event.nativeEvent.isComposing ||
                event.nativeEvent.keyCode === 229
              )
                return;
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();
                void submit(event.ctrlKey || event.metaKey);
              }
            }}
          />
        }
        headerActions={
          input.paths.length ? (
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
            </HStack>
          ) : undefined
        }
        footerActions={
          <HStack gap={1}>
            <AttachmentMenu
              disabled={!target.session}
              attach={images.attach}
              openSkills={() => setSkillsOpen(true)}
              maxFiles={capabilities.data?.inputImages ?? 8}
              maxSize={Number(capabilities.data?.imageBytes ?? 67108864)}
            />
            <ComposerActions target={target} teamEnabled={!!state.data?.team} busy={busy} />
          </HStack>
        }
        sendActions={
          task ? <ComposerQueueAction disabled={busy || !hasInput} submit={submit} /> : undefined
        }
        sendButton={
          <Tooltip
            content={showStop ? t('stop') : t('sendHint').replace('{shortcut}', queueShortcut)}
          >
            <ChatSendButton
              aria-labelledby={sendLabel}
              className="send-button"
              isStopShown={showStop}
              isDisabled={
                busy || !target.session || (showStop ? task.status === 'canceling' : !hasInput)
              }
              onSend={() => {
                void submit();
              }}
              onStop={() => {
                if (!task || busy) return;
                setBusy(true);
                void control(target.workspace, task.id, 'cancel')
                  .catch(reportError)
                  .finally(() => setBusy(false));
              }}
            />
          </Tooltip>
        }
      />
      {showStatus && <ExecutionStatus target={target} />}
      {skillsOpen && (
        <SkillPicker
          target={target}
          close={() => {
            setSkillsOpen(false);
          }}
        />
      )}
    </VStack>
  );
}
