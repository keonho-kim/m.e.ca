// Purpose: Resolve common language and framework files to one display and syntax identity.
// Pattern: Declarative file-type catalog with explicit filename precedence.
// Usage: Code language selection and Material file icons share this catalog.
// Related: webui/src/shared/lib/code-language/index.ts, webui/src/shared/ui/file-icon/index.tsx.
export const fileTypes = {
  typescript: ['TypeScript', 'typescript', 'code'],
  javascript: ['JavaScript', 'javascript', 'javascript'],
  react: ['React', 'tsx', 'framework'],
  vue: ['Vue', 'vue', 'framework'],
  angular: ['Angular', 'typescript', 'framework'],
  svelte: ['Svelte', 'svelte', 'framework'],
  html: ['HTML', 'html', 'html'],
  css: ['CSS', 'css', 'css'],
  sass: ['Sass', 'scss', 'css'],
  less: ['Less', 'less', 'css'],
  json: ['JSON', 'json', 'data'],
  yaml: ['YAML', 'yaml', 'data'],
  toml: ['TOML', 'toml', 'data'],
  markdown: ['Markdown', 'markdown', 'document'],
  python: ['Python', 'python', 'code'],
  go: ['Go', 'go', 'code'],
  rust: ['Rust', 'rust', 'code'],
  java: ['Java', 'java', 'coffee'],
  kotlin: ['Kotlin', 'kotlin', 'coffee'],
  c: ['C', 'c', 'code'],
  cpp: ['C++', 'cpp', 'code'],
  csharp: ['C#', 'csharp', 'code'],
  swift: ['Swift', 'swift', 'code'],
  dart: ['Dart', 'dart', 'code'],
  ruby: ['Ruby', 'ruby', 'code'],
  php: ['PHP', 'php', 'php'],
  shell: ['Shell', 'bash', 'terminal'],
  sql: ['SQL', 'sql', 'database'],
  graphql: ['GraphQL', 'graphql', 'schema'],
  docker: ['Docker', 'dockerfile', 'terminal'],
  terraform: ['Terraform', 'hcl', 'schema'],
  xml: ['XML', 'xml', 'code'],
} as const;
export type FileType = keyof typeof fileTypes;
const extensions: Readonly<Record<string, FileType>> = {
  ts: 'typescript',
  mts: 'typescript',
  cts: 'typescript',
  js: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
  jsx: 'react',
  tsx: 'react',
  vue: 'vue',
  svelte: 'svelte',
  html: 'html',
  htm: 'html',
  css: 'css',
  scss: 'sass',
  sass: 'sass',
  less: 'less',
  json: 'json',
  jsonc: 'json',
  yaml: 'yaml',
  yml: 'yaml',
  toml: 'toml',
  md: 'markdown',
  markdown: 'markdown',
  py: 'python',
  pyw: 'python',
  go: 'go',
  rs: 'rust',
  java: 'java',
  kt: 'kotlin',
  kts: 'kotlin',
  c: 'c',
  h: 'c',
  cc: 'cpp',
  cpp: 'cpp',
  cxx: 'cpp',
  hpp: 'cpp',
  cs: 'csharp',
  swift: 'swift',
  dart: 'dart',
  rb: 'ruby',
  php: 'php',
  sh: 'shell',
  bash: 'shell',
  zsh: 'shell',
  sql: 'sql',
  graphql: 'graphql',
  gql: 'graphql',
  tf: 'terraform',
  tfvars: 'terraform',
  xml: 'xml',
  svg: 'xml',
};
export function fileType(path: string): FileType | undefined {
  const name = path.slice(path.lastIndexOf('/') + 1).toLowerCase();
  if (name === 'angular.json' || /\.component\.ts$/.test(name)) return 'angular';
  if (
    name === 'dockerfile' ||
    name.startsWith('dockerfile.') ||
    /^(docker-)?compose\.ya?ml$/.test(name)
  )
    return 'docker';
  if (name === 'makefile' || name === '.bashrc' || name === '.zshrc') return 'shell';
  const extension = name.slice(name.lastIndexOf('.') + 1);
  return Object.hasOwn(extensions, extension) ? extensions[extension] : undefined;
}
