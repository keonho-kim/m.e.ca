// Purpose: Define the validated model form values used by the controller and Astryx fields.
// Pattern: Zod boundary schema.
// Usage: ModelForm validates saves; ModelFields binds the same typed fields.
// Related: webui/src/pages/workspace/ui/settings/model-form.tsx, webui/src/pages/workspace/ui/settings/model-fields.tsx.
import * as z from 'zod/mini';
export const modelSchema = z.object({
  provider: z.string().check(z.minLength(1)),
  endpoint: z.url().check(z.refine((value) => /^https?:\/\//i.test(value))),
  model: z.string().check(z.trim(), z.minLength(1)),
  credential: z.string().check(
    z.maxLength(2048),
    z.refine((v) => !/[\r\n\0]/.test(v))
  ),
  thinking: z.enum(['default', 'on', 'off']),
  effort: z.string(),
});
export type ModelValues = z.infer<typeof modelSchema>;
