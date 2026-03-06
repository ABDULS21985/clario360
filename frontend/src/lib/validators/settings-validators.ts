import { z } from 'zod';

const nameRegex = /^[a-zA-Z\s'-]+$/;

export const profileFormSchema = z.object({
  first_name: z
    .string()
    .min(2, 'At least 2 characters')
    .max(100, 'Maximum 100 characters')
    .regex(nameRegex, 'Only letters, spaces, hyphens, and apostrophes'),
  last_name: z
    .string()
    .min(2, 'At least 2 characters')
    .max(100, 'Maximum 100 characters')
    .regex(nameRegex, 'Only letters, spaces, hyphens, and apostrophes'),
});

export const changePasswordSchema = z
  .object({
    current_password: z.string().min(1, 'Required'),
    new_password: z
      .string()
      .min(12, 'At least 12 characters')
      .max(128, 'Maximum 128 characters')
      .regex(/[A-Z]/, 'Requires uppercase letter')
      .regex(/[a-z]/, 'Requires lowercase letter')
      .regex(/[0-9]/, 'Requires number')
      .regex(/[^a-zA-Z0-9]/, 'Requires special character'),
    confirm_password: z.string().min(1, 'Required'),
  })
  .refine((d) => d.new_password === d.confirm_password, {
    message: 'Passwords do not match',
    path: ['confirm_password'],
  })
  .refine((d) => d.new_password !== d.current_password, {
    message: 'New password must differ from current password',
    path: ['new_password'],
  });

export const createApiKeySchema = z.object({
  name: z
    .string()
    .min(1, 'Required')
    .max(100, 'Maximum 100 characters'),
  permissions: z.array(z.string()).min(1, 'At least one permission required'),
  expires_at: z.string().nullable().optional(),
  no_expiry: z.boolean().default(false),
});

export type ProfileFormData = z.infer<typeof profileFormSchema>;
export type ChangePasswordFormData = z.infer<typeof changePasswordSchema>;
export type CreateApiKeyFormData = z.infer<typeof createApiKeySchema>;
