// Purpose: Bind validated model values to official Astryx form fields and autocomplete.
// Pattern: Controlled React Hook Form field composition.
// Usage: ModelForm supplies provider metadata, current capabilities and native mutation state.
// Related: webui/src/pages/workspace/ui/settings/model-form.tsx, webui/src/pages/workspace/model/model-values.ts.

import { FormLayout } from '@astryxdesign/core/FormLayout';
import { Selector } from '@astryxdesign/core/Selector';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Typeahead } from '@astryxdesign/core/Typeahead';
import { useMemo } from 'react';
import { Controller, type UseFormReturn } from 'react-hook-form';
import type { ModelValues } from '@/pages/workspace/model/model-values';
import type { ModelCandidateJson, ProviderJson, SnapshotJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeValidation } from '@/shared/lib/errors';
export function ModelFields({
  form,
  providers,
  profile,
  candidate,
  busy,
  onProvider,
  discovering,
  loadModels,
}: {
  form: UseFormReturn<ModelValues>;
  providers: ProviderJson[];
  profile: SnapshotJson['model'];
  candidate: ModelCandidateJson | undefined;
  busy: boolean;
  onProvider: (id: string) => void;
  discovering: boolean;
  loadModels: () => Promise<ModelCandidateJson[]>;
}) {
  const t = useLocale();
  const effortLevels = candidate?.reasoningLevels ?? profile?.reasoningLevels ?? [];
  const source = useMemo(() => {
    let catalog: ModelCandidateJson[] | undefined;
    let entries: { id: string; label: string; search: string }[] = [];
    const items = async () => {
      const models = await loadModels();
      if (catalog !== models) {
        catalog = models;
        entries = models.flatMap((value) =>
          value.id ? [{ id: value.id, label: value.id, search: value.id.toLowerCase() }] : []
        );
      }
      return entries;
    };
    return {
      bootstrap: items,
      search: async (query: string) => {
        const normalized = query.toLowerCase();
        return (await items()).filter((item) => item.search.includes(normalized));
      },
    };
  }, [loadModels]);
  const error = (key: keyof ModelValues) => {
    const message = form.formState.errors[key]?.message;
    return message ? { type: 'error' as const, message: describeValidation(key, t) } : undefined;
  };
  return (
    <FormLayout direction="vertical">
      <Controller
        control={form.control}
        name="provider"
        render={({ field }) => (
          <Selector
            label={t('provider')}
            value={field.value}
            isDisabled={busy}
            options={providers.map((v) => ({ value: v.id ?? '', label: v.name ?? v.id ?? '' }))}
            onChange={onProvider}
            status={error('provider')}
          />
        )}
      />
      <Controller
        control={form.control}
        name="endpoint"
        render={({ field }) => (
          <TextInput
            ref={field.ref}
            htmlName={field.name}
            type="url"
            inputMode="url"
            isRequired
            hasAutoCorrect={false}
            hasSpellCheck={false}
            autoCapitalize="none"
            label={t('endpoint')}
            value={field.value}
            isDisabled={busy}
            autoComplete="off"
            onChange={(value) => {
              if (value !== field.value) {
                form.setValue('credential', '', { shouldDirty: true });
                form.setValue('model', '', { shouldDirty: true });
              }
              field.onChange(value);
            }}
            onBlur={field.onBlur}
            status={error('endpoint')}
          />
        )}
      />
      <Controller
        control={form.control}
        name="credential"
        render={({ field }) => (
          <TextInput
            ref={field.ref}
            htmlName={field.name}
            label={t('apiKey')}
            value={field.value}
            type="password"
            isDisabled={busy}
            autoComplete="new-password"
            onChange={field.onChange}
            onBlur={field.onBlur}
            status={error('credential')}
            description={t(profile?.credentialSaved ? 'keySaved' : 'keyEmpty')}
          />
        )}
      />
      <Controller
        control={form.control}
        name="model"
        render={({ field }) => (
          <Typeahead
            ref={field.ref}
            isRequired
            onBlur={field.onBlur}
            label={t('modelId')}
            value={
              field.value
                ? { id: field.value, label: field.value, search: field.value.toLowerCase() }
                : null
            }
            searchSource={source}
            onChange={(item) => field.onChange(item?.id ?? '')}
            onChangeQuery={field.onChange}
            isDisabled={busy}
            status={error('model')}
            hasEntriesOnFocus
            description={discovering ? t('loading') : undefined}
          />
        )}
      />
      <Controller
        control={form.control}
        name="thinking"
        render={({ field }) => (
          <Selector
            label={t('thinking')}
            value={field.value}
            isDisabled={busy}
            options={[
              { value: 'default', label: t('default') },
              { value: 'on', label: t('on') },
              {
                value: 'off',
                label: t('off'),
                disabled: (candidate?.reasoningOff ?? profile?.reasoningOff) === false,
              },
            ]}
            onChange={field.onChange}
          />
        )}
      />
      <Controller
        control={form.control}
        name="effort"
        render={({ field }) => (
          <Selector
            label={t('effort')}
            value={field.value}
            isDisabled={busy}
            options={[
              { value: '', label: t('default') },
              ...effortLevels.map((value) => ({ value, label: value })),
              ...(field.value && !effortLevels.includes(field.value)
                ? [{ value: field.value, label: field.value }]
                : []),
            ]}
            onChange={field.onChange}
          />
        )}
      />
    </FormLayout>
  );
}
