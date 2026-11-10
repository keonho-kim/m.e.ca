// Purpose: Select a skill for the current input without leaving the work area.
// Pattern: Focus-owned Astryx dialog around the shared skill browser.
// Usage: The composer attachment menu opens this dialog and restores editor focus on close.
// Related: webui/src/pages/workspace/ui/skills/skills.tsx, webui/src/pages/workspace/ui/composer/composer.tsx.
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Spinner } from '@astryxdesign/core/Spinner';
import { lazy, Suspense } from 'react';
import type { Target } from '@/pages/workspace/model/store';
import { useLocale } from '@/shared/i18n';

const Skills = lazy(() => import('@/pages/workspace/ui/skills/skills'));
export function SkillPicker({ target, close }: { target: Target; close: () => void }) {
  const t = useLocale();
  return (
    <Dialog
      isOpen
      onOpenChange={(open) => {
        if (!open) close();
      }}
      width={1040}
    >
      <Layout
        height="auto"
        padding={0}
        header={
          <VStack padding={4} gap={0}>
            <DialogHeader title={t('skills')} onOpenChange={close} />
          </VStack>
        }
        content={
          <LayoutContent padding={0} isScrollable={false}>
            <VStack height="60dvh" gap={0} className="min-h-0">
              <Suspense fallback={<Spinner label={t('loading')} />}>
                <Skills target={target} onInsert={close} />
              </Suspense>
            </VStack>
          </LayoutContent>
        }
      />
    </Dialog>
  );
}
