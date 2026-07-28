import { ImageGeneration, ImageGenerationStatus } from '../types';
import { Images } from '@/client';
import type { ImageGenerationResponse } from '@/client';
import { generatedData } from '../generatedClient';

export async function generateArticleImage(
  prompt: string | undefined, 
  articleId: number | undefined,
  generatePrompt: boolean = false,
): Promise<{ 
  success: boolean, 
  generationRequestId: string,
}> {
  if (!articleId) {
    return { success: false, generationRequestId: "" };
  }

  try {
    const result = await generatedData<{ request_id: string }>(
      Images.generateImage({
        body: {
          prompt: prompt ?? '',
          article_id: String(articleId),
          generate_prompt: generatePrompt,
        },
      }),
    );
    return { 
      success: true, 
      generationRequestId: result.request_id 
    };
  } catch (error) {
    console.error(error);
    return { success: false, generationRequestId: "" };
  }
}

export async function getImageGeneration(requestId: string): Promise<ImageGeneration | null> {
  try {
    const value = await generatedData<ImageGenerationResponse>(
      Images.getImageGeneration({ path: { requestId } }),
    );
    return {
      id: Number(value.id),
      created_at: value.created_at ? Date.parse(value.created_at) : 0,
      updated_at: value.completed_at ? Date.parse(value.completed_at) : 0,
      prompt: value.prompt,
      provider: value.provider,
      model: value.model,
      request_id: value.request_id,
      output_url: value.output_url || undefined,
      storage_key: value.file_index_id || undefined,
    };
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    console.error(error);
    return null;
  }
}

export async function getImageGenerationStatus(requestId: string): Promise<ImageGenerationStatus> {
  try {
    return generatedData<ImageGenerationStatus>(
      Images.getImageGenerationStatus({ path: { requestId } }),
    );
  } catch (error) {
    console.error(error);
    return { accepted: false, requestId, outputUrl: "" };
  }
}
