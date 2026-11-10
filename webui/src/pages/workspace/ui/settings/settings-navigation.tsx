// Purpose: Present the settings sections as a desktop list or compact top selector.
// Pattern: Responsive Astryx navigation composition.
// Usage: Settings owns the selected section and shortcut actions.
// Related: webui/src/pages/workspace/ui/settings/settings.tsx, webui/src/shared/i18n/index.ts.
import { Icon } from '@astryxdesign/core/Icon';
import { List, ListItem } from '@astryxdesign/core/List';
import { Selector } from '@astryxdesign/core/Selector';
import { useLocale } from '@/shared/i18n';

type Section = 'model' | 'execution' | 'general';
const sections = [
  { id: 'model', icon: 'meca:package' },
  { id: 'execution', icon: 'meca:play' },
  { id: 'general', icon: 'meca:sliders' },
] as const;

export function SettingsNavigation({
  compact = false,
  tab,
  onChange,
  shortcut,
}: {
  compact?: boolean;
  tab: Section;
  onChange: (value: Section) => void;
  shortcut: (value: 'files' | 'skills') => void;
}) {
  const t = useLocale();
  if (compact)
    return (
      <Selector
        label={t('settingsSection')}
        value={tab}
        options={[
          ...sections.map(({ id }) => ({ value: id, label: t(id) })),
          { value: 'files', label: t('openFiles') },
          { value: 'skills', label: t('openSkills') },
        ]}
        onChange={(value) => {
          if (value === 'files' || value === 'skills') shortcut(value);
          else if (value === 'model' || value === 'execution' || value === 'general')
            onChange(value);
        }}
      />
    );
  return (
    <List aria-label={t('settings')}>
      {sections.map(({ id, icon }) => (
        <ListItem
          key={id}
          label={t(id)}
          startContent={<Icon icon={icon} />}
          isSelected={tab === id}
          onClick={() => onChange(id)}
        />
      ))}
      <ListItem
        label={t('openFiles')}
        startContent={<Icon icon={'meca:folder-open'} />}
        onClick={() => shortcut('files')}
      />
      <ListItem
        label={t('openSkills')}
        startContent={<Icon icon={'meca:book'} />}
        onClick={() => shortcut('skills')}
      />
    </List>
  );
}
