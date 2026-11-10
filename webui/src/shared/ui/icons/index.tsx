// Purpose: Add MECA-specific Lucide icons to Astryx Neutral's built-in icon registry.
// Pattern: Theme icon-registry extension.
// Usage: The app theme adds these names while inheriting Neutral's common icons.
// Related: webui/src/app/styles/theme.ts.
import type { IconEntry } from '@astryxdesign/core/Icon';
import {
  ArrowLeft,
  BookOpen,
  File,
  Folder,
  FolderOpen,
  FolderPlus,
  ImagePlus,
  Layers,
  ListOrdered,
  MessageSquare,
  Monitor,
  Moon,
  Package,
  PanelLeft,
  Pause,
  Play,
  Plus,
  RefreshCw,
  Settings,
  SlidersHorizontal,
  SquarePen,
  Sun,
  Trash2,
  Users,
} from 'lucide-react';

const iconProps = { size: '1em', 'aria-hidden': true as const };

export const mecaIcons = {
  'meca:pause': <Pause {...iconProps} />,
  'meca:play': <Play {...iconProps} />,
  'meca:file': <File {...iconProps} />,
  'meca:folder': <Folder {...iconProps} />,
  'meca:back': <ArrowLeft {...iconProps} />,
  'meca:book': <BookOpen {...iconProps} />,
  'meca:folder-open': <FolderOpen {...iconProps} />,
  'meca:layers': <Layers {...iconProps} />,
  'meca:message': <MessageSquare {...iconProps} />,
  'meca:sidebar': <PanelLeft {...iconProps} />,
  'meca:refresh': <RefreshCw {...iconProps} />,
  'meca:queue': <ListOrdered {...iconProps} />,
  'meca:add': <Plus {...iconProps} />,
  'meca:users': <Users {...iconProps} />,
  'meca:package': <Package {...iconProps} />,
  'meca:sliders': <SlidersHorizontal {...iconProps} />,
  'meca:folder-add': <FolderPlus {...iconProps} />,
  'meca:image': <ImagePlus {...iconProps} />,
  'meca:settings': <Settings {...iconProps} />,
  'meca:monitor': <Monitor {...iconProps} />,
  'meca:moon': <Moon {...iconProps} />,
  'meca:sun': <Sun {...iconProps} />,
  'meca:edit': <SquarePen {...iconProps} />,
  'meca:delete': <Trash2 {...iconProps} />,
} satisfies Record<`meca:${string}`, IconEntry>;
