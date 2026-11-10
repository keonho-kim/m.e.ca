// Purpose: Own one cancelable model-catalog request and share its result within a settings draft.
// Pattern: Scoped asynchronous metadata source.
// Usage: Astryx Typeahead awaits load for bootstrap and search; ModelForm can refresh explicitly.
// Related: webui/src/pages/workspace/ui/settings/model-form.tsx, webui/src/pages/workspace/ui/settings/model-fields.tsx.
import { useCallback, useEffect, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import type { ModelValues } from '@/pages/workspace/model/model-values';
import type { Target } from '@/pages/workspace/model/store';
import { call, type ModelCandidateJson } from '@/shared/api';
import { describeError, failure } from '@/shared/lib/errors';

export function useModelCandidates(target: Target) {
  const [pending, setPending] = useState(false),
    [error, setError] = useState('');
  const values = useRef<ModelCandidateJson[]>([]);
  const loaded = useRef(false);
  const request = useRef<{
    controller: AbortController;
    promise: Promise<ModelCandidateJson[]>;
  } | null>(null);
  const reset = useCallback(() => {
    request.current?.controller.abort();
    request.current = null;
    values.current = [];
    loaded.current = false;
    setError('');
    setPending(false);
  }, []);
  useEffect(() => {
    reset();
    return () => request.current?.controller.abort();
  }, [target.workspace, target.session, reset]);
  const load = useCallback(
    (value: ModelValues, refresh = false): Promise<ModelCandidateJson[]> => {
      if (!refresh && loaded.current) return Promise.resolve(values.current);
      if (!refresh && request.current) return request.current.promise;
      request.current?.controller.abort();
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
            target.workspace,
            {
              discoverModels: {
                sessionId: target.session,
                provider: value.provider,
                endpoint: value.endpoint,
                credential: value.credential,
              },
            },
            controller.signal
          );
          controller.signal.throwIfAborted();
          values.current = reply.models?.items ?? [];
          loaded.current = true;
          return values.current;
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
      request.current = { controller, promise };
      return promise;
    },
    [target.workspace, target.session]
  );
  return { pending, error, load, reset };
}
