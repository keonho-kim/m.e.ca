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
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { useQuery } from '@tanstack/react-query';
import {
  lazy,
  type Ref,
  Suspense,
  useCallback,
  useEffect,
  useId,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import { reportError } from '@/entities/server-session';
import { capabilitiesOptions, stateOptions } from '@/pages/workspace/api/queries';
import { command, settings, view } from '@/pages/workspace/model/commands';
import { send } from '@/pages/workspace/model/execution';
import { selectReference } from '@/pages/workspace/model/file-references';
import { waitSelections } from '@/pages/workspace/model/reference-queue';
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
import { ExecutionStatus } from '@/pages/workspace/ui/composer/execution-status';
import { Queue } from '@/pages/workspace/ui/composer/queue';
import { SkillPicker } from '@/pages/workspace/ui/skills/skill-picker';
import { control } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { failure } from '@/shared/lib/errors';
import { ReferenceToken, referenceToken } from '@/shared/ui/reference-token';

const ComposerImages = lazy(() =>
  import('@/pages/workspace/ui/composer/composer-images').then((module) => ({
    default: module.ComposerImages,
  }))
);

export type ComposerHandle = { insertSkill: (mention: string) => void };
export function Composer({
  target,
  showStatus,
  ref,
}: {
  target: Target;
  showStatus: boolean;
  ref: Ref<ComposerHandle>;
}) {
  const t = useLocale(),
    input = useWorkspaceStore((s) => s.inputs[targetKey(target)]) ?? emptyInput();
  const task = useWorkspaceStore((state) => activeTask(target, state.tasks));
  const hasInput = !!(input.text.trim() || input.files.length);
  const showStop = !!task && !hasInput;
  const queueShortcut = /Mac|iPad|iPhone/.test(navigator.userAgent) ? '⌘↵' : 'Ctrl+↵';
  const editor = useRef<ChatComposerInputHandle>(null);
  const insertSkill = (mention: string) => {
    editor.current?.focus();
    editor.current?.insertToken(referenceToken(`#${mention}`));
  };
  useImperativeHandle(ref, () => ({ insertSkill }), []);
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
        if (useWorkspaceStore.getState().fileDirty) throw failure('draft_changed');
        if (!state.data?.model?.model) {
          settings('model', target);
          return;
        }
        await waitSelections(target);
        const captured = useWorkspaceStore.getState().inputs[targetKey(target)] ?? input;
        if (!capabilities.data) throw failure('invalid_response');
        checkSubmissionLimits(captured, capabilities.data);
        if (await send(target, captured, reserve)) view('chat');
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
      {!!input.files.length && (
        <Suspense fallback={null}>
          <ComposerImages target={target} onEmpty={focusInput} />
        </Suspense>
      )}
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
                <HStack key={path} gap={0} className="min-w-0 max-w-full">
                  <ReferenceToken value={path} />
                  <IconButton
                    size="sm"
                    variant="ghost"
                    label={`${t('removeReference')} · ${path}`}
                    icon={<Icon icon="meca:close" />}
                    onClick={() => {
                      void selectReference(target, path, false).catch(reportError);
                    }}
                  />
                </HStack>
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
          onInsert={insertSkill}
          close={() => {
            setSkillsOpen(false);
          }}
        />
      )}
    </VStack>
  );
}
