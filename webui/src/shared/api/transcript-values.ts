// Purpose: Validate bounded transcript payload values without copying response bodies.
// Pattern: Allocation-light transport assertions.
// Usage: The transcript parser validates message, notice and completion bodies here.
// Related: webui/src/shared/api/transcript.ts, api/meca/v1/events.proto.
import { failure } from '@/shared/lib/errors';

export function object(value: unknown): asserts value is Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value))
    throw failure('invalid_response');
}
export function only(value: Record<string, unknown>, keys: readonly string[]) {
  if (Object.keys(value).some((key) => !keys.includes(key))) throw failure('invalid_response');
}
export function text(value: unknown, max = 1024 * 1024) {
  if (value !== undefined && (typeof value !== 'string' || value.length > max))
    throw failure('invalid_response');
}
export function integer(value: unknown, max = 9223372036854775807n) {
  if (
    value !== undefined &&
    (typeof value !== 'string' || !/^(0|[1-9]\d{0,19})$/.test(value) || BigInt(value) > max)
  )
    throw failure('invalid_response');
}
export function boolean(value: unknown) {
  if (value !== undefined && typeof value !== 'boolean') throw failure('invalid_response');
}
export function statistics(value: unknown) {
  if (value === undefined) return;
  object(value);
  only(value, ['concurrent', 'elapsedMillis', 'generationMillis', 'usage']);
  boolean(value.concurrent);
  integer(value.elapsedMillis);
  integer(value.generationMillis);
  if (value.usage !== undefined) {
    object(value.usage);
    only(value.usage, ['input', 'output', 'cost']);
    integer(value.usage.input);
    integer(value.usage.output);
    if (
      value.usage.cost !== undefined &&
      (typeof value.usage.cost !== 'number' ||
        !Number.isFinite(value.usage.cost) ||
        value.usage.cost < 0)
    )
      throw failure('invalid_response');
  }
}
export function completion(value: unknown) {
  object(value);
  only(value, ['status', 'statistics']);
  if (!['COMPLETED', 'FAILED', 'INTERRUPTED', 'DENIED', 'EXHAUSTED'].includes(String(value.status)))
    throw failure('invalid_response');
  statistics(value.statistics);
}
export function team(value: unknown) {
  if (value === undefined) return;
  object(value);
  only(value, ['replyTo', 'replyAuthor', 'mentions', 'basedOn', 'paths', 'contract', 'revision']);
  integer(value.replyTo);
  integer(value.basedOn);
  integer(value.revision);
  text(value.replyAuthor, 128);
  text(value.contract);
  for (const key of ['mentions', 'paths'])
    if (value[key] !== undefined) {
      const items = value[key];
      if (!Array.isArray(items) || items.length > 32) throw failure('invalid_response');
      for (const item of items) {
        if (typeof item !== 'string') throw failure('invalid_response');
        text(item, 4096);
      }
    }
}
export function message(value: unknown) {
  object(value);
  only(value, ['role', 'text', 'thinking', 'incomplete', 'images', 'team']);
  if (value.role !== 'assistant' && value.role !== 'user') throw failure('invalid_response');
  text(value.text);
  text(value.thinking);
  boolean(value.incomplete);
  team(value.team);
  if (value.images !== undefined) {
    if (!Array.isArray(value.images) || value.images.length > 8) throw failure('invalid_response');
    for (const image of value.images) {
      object(image);
      if (
        image.index !== undefined &&
        (!Number.isInteger(image.index) || Number(image.index) < 0 || Number(image.index) > 7)
      )
        throw failure('invalid_response');
    }
  }
}
export function notice(value: unknown) {
  object(value);
  only(value, ['code', 'skill']);
  if (
    !['RETRYING', 'COMPACTION_STARTED', 'COMPACTION_FINISHED', 'SKILL_READ'].includes(
      String(value.code)
    )
  )
    throw failure('invalid_response');
  if (value.code === 'SKILL_READ') {
    object(value.skill);
    only(value.skill, ['id', 'name', 'source', 'revision']);
    if (
      typeof value.skill.name !== 'string' ||
      !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(value.skill.name) ||
      value.skill.name.length > 64
    )
      throw failure('invalid_response');
    text(value.skill.id, 512);
    text(value.skill.source, 32);
    text(value.skill.revision, 64);
  } else if (value.skill !== undefined) throw failure('invalid_response');
}
