import { z } from 'zod';

const nameRegex = /^[a-zA-Z\s'-]+$/;

const passwordSchema = z
  .string()
  .min(12, 'At least 12 characters')
  .max(128, 'Maximum 128 characters')
  .regex(/[A-Z]/, 'Requires uppercase letter')
  .regex(/[a-z]/, 'Requires lowercase letter')
  .regex(/[0-9]/, 'Requires number')
  .regex(/[^a-zA-Z0-9]/, 'Requires special character');

export const createUserSchema = z
  .object({
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
    email: z.string().min(1, 'Required').email('Invalid email address'),
    password: passwordSchema,
    confirm_password: z.string().min(1, 'Required'),
    roles: z.array(z.string()).min(1, 'At least one role required'),
    status: z.enum(['active', 'suspended', 'deactivated']).default('active'),
    send_welcome_email: z.boolean().default(true),
  })
  .refine((d) => d.password === d.confirm_password, {
    message: 'Passwords do not match',
    path: ['confirm_password'],
  });

export const editUserSchema = z.object({
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
  status: z.enum(['active', 'suspended', 'deactivated']),
});

export const resetPasswordEmailSchema = z.object({
  mode: z.literal('email'),
});

export const resetPasswordTempSchema = z
  .object({
    mode: z.literal('temp'),
    temp_password: passwordSchema,
    require_change: z.boolean().default(true),
  });

export type CreateUserFormData = z.infer<typeof createUserSchema>;
export type EditUserFormData = z.infer<typeof editUserSchema>;
