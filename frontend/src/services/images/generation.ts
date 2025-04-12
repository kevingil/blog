'use server'

import { ImageGeneration, ImageGenerationStatus } from '../types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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
    const response = await fetch(`${API_BASE_URL}/api/images/generate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        prompt,
        article_id: articleId,
        generate_prompt: generatePrompt,
      }),
    });

    if (!response.ok) {
      throw new Error('Failed to generate image');
    }

    const result = await response.json();
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
    const response = await fetch(`${API_BASE_URL}/api/images/${requestId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error('Failed to get image generation');
    }

    return response.json();
  } catch (error) {
    console.error(error);
    return null;
  }
}

export async function getImageGenerationStatus(requestId: string): Promise<ImageGenerationStatus> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/images/${requestId}/status`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error('Failed to get image generation status');
    }

    return response.json();
  } catch (error) {
    console.error(error);
    return { accepted: false, requestId, outputUrl: "" };
  }
}

