// Purpose: Define the validated model form values used by the controller and Astryx fields.
// Pattern: Zod boundary schema.
// Usage: ModelForm validates saves; ModelFields binds the same typed fields.
// Related: webui/src/pages/workspace/ui/settings/model-form.tsx, webui/src/pages/workspace/ui/settings/model-fields.tsx.
import { z } from 'zod';
export const modelSchema = z.object({
  provider: z.string().min(1),
  endpoint: z.string().url(),
  model: z.string().trim().min(1),
  credential: z
    .string()
    .max(2048)
    .refine((v) => !/[\r\n\0]/.test(v)),
  thinking: z.enum(['default', 'on', 'off']),
  effort: z.string(),
  remember: z.boolean(),
});
export type ModelValues = z.infer<typeof modelSchema>;
