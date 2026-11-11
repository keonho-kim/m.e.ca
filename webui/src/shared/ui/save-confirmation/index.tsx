// Purpose: Confirm explicit settings saves with shared wording and pending/error feedback.
// Pattern: Astryx confirmation dialog composition.
// Usage: Model and execution settings close their parent only after a successful write.
// Related: webui/src/pages/workspace/ui/settings/model-form.tsx, webui/src/pages/workspace/ui/settings/settings.tsx.
import { AlertDialog } from '@astryxdesign/core/AlertDialog';
import { useLocale } from '@/shared/i18n';
export function SaveConfirmation({
  open,
  busy,
  error,
  close,
  save,
}: {
  open: boolean;
  busy: boolean;
  error: string;
  close: () => void;
  save: () => void;
}) {
  const t = useLocale();
  return (
    <AlertDialog
      isOpen={open}
      title={t('saveConfirm')}
      description={error || t('saveConfirmHelp')}
      cancelLabel={t('no')}
      actionLabel={t('yes')}
      isActionLoading={busy}
      onOpenChange={(value) => {
        if (!value && !busy) close();
      }}
      onAction={save}
    />
  );
}
