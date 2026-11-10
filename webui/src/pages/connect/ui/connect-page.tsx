// Purpose: Authenticate once without persisting the server bearer token in browser storage.
// Pattern: Scoped authentication form.
// Usage: App shows Login until a browser cookie is accepted by the native API.
// Related: internal/serverapi/ui_auth.go, webui/src/shared/api/client.ts.

import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Center } from '@astryxdesign/core/Center';
import { Heading } from '@astryxdesign/core/Heading';
import { Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { VStack } from '@astryxdesign/core/VStack';
import { useState } from 'react';
import { authenticate } from '@/features/server-access';
import { useLocale } from '@/shared/i18n';
import { describeError } from '@/shared/lib/errors';

export function Login() {
  const t = useLocale(),
    [error, setError] = useState('');
  const [token, setToken] = useState(''),
    [busy, setBusy] = useState(false);
  return (
    <Center height="100dvh" padding={6}>
      <VStack maxWidth={440} width="100%" gap={6}>
        <Heading level={1} type="display-3">
          MECA
        </Heading>
        <Text color="secondary">{t('loginHelp')}</Text>
        <VStack
          as="form"
          gap={5}
          onSubmit={async (event) => {
            event.preventDefault();
            if (busy || !token.trim() || token.length > 4096) return;
            setBusy(true);
            setError('');
            try {
              await authenticate(token.trim());
              setToken('');
            } catch (error) {
              setError(describeError(error));
            } finally {
              setBusy(false);
            }
          }}
        >
          <TextInput
            label={t('token')}
            type="password"
            value={token}
            onChange={setToken}
            autoComplete="off"
            hasAutoFocus
          />
          {error && <Banner status="error" title={error} />}
          <Button label={t('login')} type="submit" variant="primary" isLoading={busy} />
        </VStack>
      </VStack>
    </Center>
  );
}
