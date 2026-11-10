// Purpose: Bind validated model values to official Astryx form fields and autocomplete.
// Pattern: Controlled React Hook Form field composition.
// Usage: ModelForm supplies provider metadata, current capabilities and native mutation state.
// Related: webui/src/pages/workspace/ui/settings/model-form.tsx, webui/src/pages/workspace/model/model-values.ts.

import { CheckboxInput } from '@astryxdesign/core/CheckboxInput';
import { FormLayout } from '@astryxdesign/core/FormLayout';
import { Selector } from '@astryxdesign/core/Selector';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Typeahead } from '@astryxdesign/core/Typeahead';
import { useMemo } from 'react';
import { Controller, type UseFormReturn } from 'react-hook-form';
import type { ModelValues } from '@/pages/workspace/model/model-values';
import type { ModelCandidateJson, ProviderJson, SnapshotJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function ModelFields({
  form,
  providers,
  profile,
  busy,
  onProvider,
  discovering,
  loadModels,
}: {
  form: UseFormReturn<ModelValues>;
  providers: ProviderJson[];
  profile: SnapshotJson['model'];
  busy: boolean;
  onProvider: (id: string) => void;
  discovering: boolean;
  loadModels: () => Promise<ModelCandidateJson[]>;
}) {
  const t = useLocale();
  const source = useMemo(() => {
    const items = async () =>
      (await loadModels()).flatMap((value) =>
        value.id ? [{ id: value.id, label: value.id }] : []
      );
    return {
      bootstrap: items,
      search: async (query: string) =>
        (await items()).filter((item) => item.label.toLowerCase().includes(query.toLowerCase())),
    };
  }, [loadModels]);
  const error = (key: keyof ModelValues) => {
    const message = form.formState.errors[key]?.message;
    return message ? { type: 'error' as const, message } : undefined;
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
            label={t('modelId')}
            value={field.value ? { id: field.value, label: field.value } : null}
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
              { value: 'off', label: t('off'), disabled: profile?.reasoningOff === false },
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
              ...(profile?.reasoningLevels ?? []).map((value) => ({ value, label: value })),
            ]}
            onChange={field.onChange}
          />
        )}
      />
      <Controller
        control={form.control}
        name="remember"
        render={({ field }) => (
          <CheckboxInput
            label={t('remember')}
            value={field.value}
            isDisabled={busy}
            onChange={field.onChange}
          />
        )}
      />
    </FormLayout>
  );
}
