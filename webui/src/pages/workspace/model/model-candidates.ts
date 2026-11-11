// Purpose: Own one cancelable model-catalog request and share its result within a settings draft.
// Pattern: Scoped asynchronous metadata source.
// Usage: Astryx Typeahead awaits load on focus and while searching.
// Related: webui/src/pages/workspace/ui/settings/model-form.tsx, webui/src/pages/workspace/ui/settings/model-fields.tsx.
import { useCallback, useEffect, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import type { ModelValues } from '@/pages/workspace/model/model-values';
import { call, type ModelCandidateJson } from '@/shared/api';
import { describeError, failure } from '@/shared/lib/errors';

type CatalogConnection = Pick<ModelValues, 'provider' | 'endpoint' | 'credential'>;
function sameConnection(a: CatalogConnection, b: CatalogConnection) {
  return a.provider === b.provider && a.endpoint === b.endpoint && a.credential === b.credential;
}

export function useModelCandidates() {
  const [pending, setPending] = useState(false),
    [error, setError] = useState('');
  const [models, setModels] = useState<ModelCandidateJson[]>([]);
  const cached = useRef<{ connection: CatalogConnection; models: ModelCandidateJson[] } | null>(
    null
  );
  const request = useRef<{
    connection: CatalogConnection;
    controller: AbortController;
    promise: Promise<ModelCandidateJson[]>;
  } | null>(null);
  const reset = useCallback(() => {
    request.current?.controller.abort();
    request.current = null;
    cached.current = null;
    setModels([]);
    setError('');
    setPending(false);
  }, []);
  useEffect(() => () => request.current?.controller.abort(), []);
  const load = useCallback((value: CatalogConnection): Promise<ModelCandidateJson[]> => {
    if (cached.current && sameConnection(cached.current.connection, value))
      return Promise.resolve(cached.current.models);
    if (request.current && sameConnection(request.current.connection, value))
      return request.current.promise;
    request.current?.controller.abort();
    cached.current = null;
    const controller = new AbortController();
    setPending(true);
    setError('');
    const promise = Promise.resolve().then(async () => {
      try {
        controller.signal.throwIfAborted();
        const url = URL.parse(value.endpoint);
        if (!value.provider || !url || !['http:', 'https:'].includes(url.protocol))
          throw failure('invalid_model_connection');
        const reply = await call(
          '',
          {
            discoverModels: {
              provider: value.provider,
              endpoint: value.endpoint,
              credential: value.credential,
            },
          },
          controller.signal
        );
        controller.signal.throwIfAborted();
        const result = reply.models?.items ?? [];
        cached.current = { connection: { ...value }, models: result };
        setModels(result);
        return result;
      } catch (error) {
        if (!controller.signal.aborted) {
          setError(describeError(error));
          reportError(error);
        }
        return [];
      } finally {
        if (request.current?.controller === controller) {
          request.current = null;
          setPending(false);
        }
      }
    });
    request.current = { connection: { ...value }, controller, promise };
    return promise;
  }, []);
  return { pending, error, models, load, reset };
}
