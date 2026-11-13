// Purpose: Preserve the browser's unsaved-work warning when a lazy feature fails.
// Pattern: Small presentation flag with application-owned beforeunload lifetime.
// Usage: Workspace publishes dirty state; Root installs the document listener.
// Related: webui/src/app/main.tsx, webui/src/pages/workspace/ui/workspace-page.tsx.
let unsaved = false;
type RecoveryFile = { name: string; text: string; discard: () => void };
let file: RecoveryFile | null = null;
const listeners = new Set<() => void>();
export function recoveryFile() {
  return file;
}
export function observeRecoveryFile(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
export function keepRecoveryFile(name: string, text: string, discard: () => void) {
  file = { name, text, discard };
  for (const listener of listeners) listener();
}
export function discardRecoveryFile() {
  const previous = file;
  file = null;
  previous?.discard();
  for (const listener of listeners) listener();
}
export function downloadRecoveryFile() {
  if (!file) return;
  const url = URL.createObjectURL(new Blob([file.text], { type: 'text/plain;charset=utf-8' }));
  const link = document.createElement('a');
  link.href = url;
  link.download = file.name.split('/').at(-1) || 'unsaved.txt';
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}
export function setReloadWarning(value: boolean) {
  if (unsaved === value) return;
  unsaved = value;
  for (const listener of listeners) listener();
}
export function watchReload() {
  let attached = false;
  const warn = (event: BeforeUnloadEvent) => {
    if (unsaved || file) event.preventDefault();
  };
  const update = () => {
    const required = unsaved || file !== null;
    if (required === attached) return;
    window[required ? 'addEventListener' : 'removeEventListener']('beforeunload', warn);
    attached = required;
  };
  listeners.add(update);
  update();
  return () => {
    listeners.delete(update);
    window.removeEventListener('beforeunload', warn);
  };
}
