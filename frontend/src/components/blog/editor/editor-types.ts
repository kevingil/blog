import * as z from 'zod';

export const DEFAULT_IMAGE_PROMPT = [
  "A modern, minimalist illustration",
  "A vibrant, colorful scene",
  "A professional business setting",
  "A natural landscape",
  "An abstract design"
];

export const articleSchema = z.object({
  title: z.string().max(200, 'Title must be at most 200 characters'),
  content: z.string(),
  image_url: z.union([z.string().url(), z.literal('')]).optional(),
  tags: z.array(z.string()),
  external_url: z.string().optional(),
}).superRefine((value, ctx) => {
  const external = value.external_url?.trim() ?? '';
  if (external && !/^https?:\/\//i.test(external)) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['external_url'],
      message: 'Enter a full http or https link.',
    });
  }
});

export type ArticleFormData = z.infer<typeof articleSchema>;
