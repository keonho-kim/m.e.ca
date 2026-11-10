// Purpose: Edit and diagnose one explicitly targeted conversation's model configuration.
// Pattern: React Hook Form with Zod validation and native API mutations.
// Usage: Settings keeps one model draft until it is saved or discarded.
// Related: webui/src/pages/workspace/ui/settings/settings.tsx, api/meca/v1/control.proto.

import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { FieldStatus } from '@astryxdesign/core/FieldStatus';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { reportError } from '@/entities/server-session';
import { changed, stateOptions } from '@/pages/workspace/api/queries';
import { useModelCandidates } from '@/pages/workspace/model/model-candidates';
import { type ModelValues, modelSchema } from '@/pages/workspace/model/model-values';
import type { Target } from '@/pages/workspace/model/store';
import { ModelFields } from '@/pages/workspace/ui/settings/model-fields';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeError } from '@/shared/lib/errors';

export function ModelForm({
  target,
  active,
  onDirty,
  onBusy,
}: {
  target: Target;
  active: boolean;
  onDirty: (dirty: boolean) => void;
  onBusy: (busy: boolean) => void;
}) {
  const t = useLocale(),
    state = useQuery({ ...stateOptions(target), enabled: !!target.session });
  const providers = useQuery({
    queryKey: ['providers'],
    queryFn: async ({ signal }) =>
      (await call('', { providers: {} }, signal)).providers?.items ?? [],
  });
  const candidates = useModelCandidates(target);
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState(''),
    [error, setError] = useState('');
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => controller.current?.abort(), []);
  useEffect(() => {
    onBusy(busy);
    return () => onBusy(false);
  }, [busy, onBusy]);
  const request = () => {
    controller.current?.abort();
    controller.current = new AbortController();
    return controller.current.signal;
  };
  const form = useForm<ModelValues>({
    resolver: zodResolver(modelSchema),
    defaultValues: {
      provider: 'ollama',
      endpoint: 'http://127.0.0.1:11434',
      model: '',
      credential: '',
      thinking: 'default',
      effort: '',
      remember: false,
    },
  });
  const { isDirty } = form.formState;
  useEffect(() => {
    onDirty(isDirty);
  }, [isDirty, onDirty]);
  useEffect(() => {
    if (!state.data || isDirty) return;
    const model = state.data.model;
    if (model)
      form.reset({
        provider: model.provider ?? 'ollama',
        endpoint: model.endpoint ?? '',
        model: model.model ?? '',
        credential: '',
        thinking: model.thinking === undefined ? 'default' : model.thinking ? 'on' : 'off',
        effort: model.reasoningEffort ?? '',
        remember: false,
      });
  }, [state.data, isDirty, form.reset]);

  const [provider, endpoint, credential] = useWatch({
    control: form.control,
    name: ['provider', 'endpoint', 'credential'],
  });
  useEffect(candidates.reset, [provider, endpoint, credential, candidates.reset]);
  const discover = () => candidates.load(form.getValues(), true);
  const loadModels = useCallback(
    () => candidates.load({ ...form.getValues(), provider, endpoint, credential }),
    [candidates.load, form, provider, endpoint, credential]
  );
  const diagnose = form.handleSubmit(async (value) => {
    setBusy(true);
    setStatus('');
    setError('');
    try {
      await call(
        target.workspace,
        {
          diagnoseModel: {
            sessionId: target.session,
            provider: value.provider,
            endpoint: value.endpoint,
            credential: value.credential,
            model: value.model,
          },
        },
        request()
      );
      setStatus(t('diagnosed'));
    } catch (error) {
      if (!controller.current?.signal.aborted) {
        setError(describeError(error));
        reportError(error);
      }
    } finally {
      setBusy(false);
    }
  });
  const save = form.handleSubmit(async (value) => {
    setBusy(true);
    setStatus('');
    setError('');
    try {
      const reply = await call(
        target.workspace,
        {
          setModel: {
            sessionId: target.session,
            provider: value.provider,
            endpoint: value.endpoint,
            model: value.model,
            credential: value.credential,
            ...(value.thinking === 'default' ? {} : { thinking: value.thinking === 'on' }),
            ...(value.effort ? { reasoningEffort: value.effort } : {}),
            remember: value.remember,
          },
        },
        request()
      );
      if (reply.state) queries.setQueryData(stateOptions(target).queryKey, reply.state);
      form.reset({ ...value, credential: '', remember: false });
      changed(target);
      setStatus(t('saved'));
    } catch (error) {
      if (!controller.current?.signal.aborted) {
        setError(describeError(error));
        reportError(error);
      }
    } finally {
      setBusy(false);
    }
  });
  return (
    <VStack
      as="form"
      hidden={!active}
      className="settings-form"
      gap={5}
      padding={6}
      onSubmit={save}
    >
      {!target.session ? (
        <Text color="secondary">{t('modelRequired')}</Text>
      ) : (
        <>
          {(error || candidates.error) && (
            <Banner status="error" title={error || candidates.error} />
          )}
          <Text type="supporting" color="secondary">
            {t('scope')} · {t('currentChat')}
          </Text>
          <ModelFields
            form={form}
            providers={providers.data ?? []}
            profile={state.data?.model}
            discovering={candidates.pending}
            loadModels={loadModels}
            busy={busy}
            onProvider={(id) => {
              form.setValue('provider', id, { shouldDirty: true });
              form.setValue('credential', '', { shouldDirty: true });
              form.setValue('model', '', { shouldDirty: true });
              form.setValue('thinking', 'default', { shouldDirty: true });
              form.setValue('effort', '', { shouldDirty: true });
              const provider = providers.data?.find((v) => v.id === id);
              if (provider?.endpoint)
                form.setValue('endpoint', provider.endpoint, { shouldDirty: true });
              candidates.reset();
            }}
          />
          <Button label={t('discover')} onClick={discover} isDisabled={busy} size="sm" />
          {status && <FieldStatus type="success" message={status} />}
          <HStack gap={3} justify="between" className="form-actions" wrap="wrap">
            <Button label={t('diagnose')} onClick={() => diagnose()} isDisabled={busy} />
            <Button label={t('save')} type="submit" variant="primary" isLoading={busy} />
          </HStack>
        </>
      )}
    </VStack>
  );
}
