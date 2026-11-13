// Purpose: Present every settings command and navigation shortcut in one scoped modal.
// Pattern: Stateful settings surface with explicit per-operation saves.
// Usage: The sidebar gear and composer commands open the same Settings component.
// Related: webui/src/pages/workspace/ui/settings/model-form.tsx, webui/src/pages/workspace/ui/settings/model-fields.tsx.

import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { useMediaQuery } from '@astryxdesign/core/hooks';
import { Layout, LayoutContent, LayoutPanel, VStack } from '@astryxdesign/core/Layout';
import { Selector } from '@astryxdesign/core/Selector';
import { Switch } from '@astryxdesign/core/Switch';
import { Text } from '@astryxdesign/core/Text';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { changed, stateOptions } from '@/pages/workspace/api/queries';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { ModelForm } from '@/pages/workspace/ui/settings/model-form';
import { SettingsNavigation } from '@/pages/workspace/ui/settings/settings-navigation';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeError } from '@/shared/lib/errors';
import { SaveConfirmation } from '@/shared/ui/save-confirmation';

export default function Settings() {
  const settings = useWorkspaceStore((s) => s.settings)!,
    t = useLocale();
  const compact = useMediaQuery('(max-width: 900px)');
  const [tab, setTab] = useState(settings.tab),
    [dirty, setDirty] = useState(false),
    [discard, setDiscard] = useState(false);
  const [busy, setBusy] = useState(false),
    [permission, setPermission] = useState('default');
  const [confirmSave, setConfirmSave] = useState(false);
  const onSaved = useCallback(() => useWorkspaceStore.getState().set({ settings: null }), []);
  const [modelBusy, setModelBusy] = useState(false),
    [error, setError] = useState('');
  const target = settings.target,
    state = useQuery({
      ...stateOptions(target),
      enabled: tab === 'execution' && !!target.workspace,
    });
  const onDirty = useCallback((value: boolean) => setDirty(value), []);
  useEffect(() => {
    setPermission(state.data?.permission ?? 'default');
  }, [state.data?.permission]);
  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
    };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [dirty]);
  const close = () => {
    if (busy || modelBusy) return;
    if (dirty) setDiscard(true);
    else useWorkspaceStore.getState().set({ settings: null });
  };
  const configure = async (value: { permission?: string; team?: boolean }, closeAfter = false) => {
    setBusy(true);
    setError('');
    try {
      await call(target.workspace, {
        setSettings: {
          ...value,
          ...(value.team === undefined ? {} : { sessionId: target.session }),
        },
      });
      changed(target);
      if (closeAfter) onSaved();
    } catch (error) {
      setError(describeError(error));
      reportError(error);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <Dialog
        isOpen
        onOpenChange={(open) => {
          if (!open) close();
        }}
        width={1040}
        purpose="form"
        className="settings-modal"
      >
        <Layout
          header={
            <VStack gap={0}>
              <DialogHeader title={t('settings')} onOpenChange={() => close()} />
              {compact && (
                <VStack paddingInline={4} paddingBlockEnd={3}>
                  <SettingsNavigation compact tab={tab} onChange={setTab} />
                </VStack>
              )}
            </VStack>
          }
          start={
            compact ? undefined : (
              <LayoutPanel width={224} padding={2}>
                <SettingsNavigation tab={tab} onChange={setTab} />
              </LayoutPanel>
            )
          }
          content={
            <LayoutContent padding={0}>
              {error && <Banner status="error" title={error} />}
              <ModelForm
                active={tab === 'model'}
                onDirty={onDirty}
                onBusy={setModelBusy}
                onSaved={onSaved}
              />
              <VStack hidden={tab !== 'execution'} padding={6} gap={5}>
                <Text type="label" weight="semibold">
                  {t('permission')} · {t('global')}
                </Text>
                <Text color="secondary">{t('permissionHelp')}</Text>
                <Selector
                  label={t('permission')}
                  value={permission}
                  onChange={setPermission}
                  isDisabled={busy}
                  options={[
                    { value: 'default', label: t('permissionDefault') },
                    { value: 'all_mighty', label: t('permissionAll') },
                  ]}
                />
                <Button
                  label={t('save')}
                  onClick={() => {
                    setError('');
                    setConfirmSave(true);
                  }}
                  isDisabled={busy}
                />
                <Text type="label" weight="semibold">
                  {t('team')} · {t('currentChat')}
                </Text>
                <Switch
                  label={t('team')}
                  description={t('teamHelp')}
                  value={!!state.data?.team}
                  isDisabled={busy || !target.session}
                  onChange={(team) => {
                    void configure({ team });
                  }}
                />
              </VStack>
            </LayoutContent>
          }
        />
      </Dialog>
      <SaveConfirmation
        open={confirmSave}
        busy={busy}
        error={error}
        close={() => setConfirmSave(false)}
        save={() => {
          void configure({ permission }, true);
        }}
      />
      {discard && (
        <AlertDialog
          isOpen={discard}
          onOpenChange={setDiscard}
          title={t('dirty')}
          description={t('unsavedModel')}
          cancelLabel={t('keepEditing')}
          actionLabel={t('discard')}
          isActionLoading={busy || modelBusy}
          onAction={() => {
            if (!busy && !modelBusy) useWorkspaceStore.getState().set({ settings: null });
          }}
        />
      )}
    </>
  );
}
