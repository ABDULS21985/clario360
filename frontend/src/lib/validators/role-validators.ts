import { z } from 'zod';

export const createRoleSchema = z.object({
  name: z
    .string()
    .min(3, 'At least 3 characters')
    .max(100, 'Maximum 100 characters'),
  description: z
    .string()
    .max(500, 'Maximum 500 characters')
    .optional()
    .default(''),
  permissions: z.array(z.string()).min(1, 'At least one permission required'),
});

export const editRoleSchema = createRoleSchema;

export type RoleFormData = z.infer<typeof createRoleSchema>;
