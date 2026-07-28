import { useState, useEffect } from 'react';
import { ArticleListItem } from '@/services/types';
import { getImageGeneration, getImageGenerationStatus } from '@/services/blog';

export function ImageLoader({ article, newImageGenerationRequestId, stagedImageUrl, setStagedImageUrl }: {
  article: ArticleListItem | null | undefined,
  newImageGenerationRequestId: string | null | undefined,
  stagedImageUrl: string | null | undefined,
  setStagedImageUrl: (url: string | null | undefined) => void
}) {
  const [imageUrl, setImageUrl] = useState<string | null>(null);

  useEffect(() => {
    const requestToFetch = newImageGenerationRequestId || article?.article.image_generation_request_id || null;
    async function fetchImageGeneration() {
      if (requestToFetch) {
        const imgGen = await getImageGeneration(requestToFetch);
        if (imgGen) {
          if (imgGen.outputUrl) {
            setImageUrl(imgGen.outputUrl);
            setStagedImageUrl(imgGen.outputUrl);
          } else {
            const status = await getImageGenerationStatus(requestToFetch);
            if (status.outputUrl) {
              setImageUrl(status.outputUrl);
              setStagedImageUrl(status.outputUrl);
            }
          }
        }
      }
    }
    fetchImageGeneration();

    if (stagedImageUrl !== undefined) {
      setImageUrl(stagedImageUrl);
    } else if (article && article.article.draft_image_url) {
      setImageUrl(article.article.draft_image_url);
    }
  }, [article, stagedImageUrl, newImageGenerationRequestId]);

  if (!article) {
    return null;
  }

  if (imageUrl) {
    return (
      <div className='flex items-center justify-center'>
        <img className='rounded-md aspect-video object-cover' src={imageUrl} alt={article.article.draft_title || ''} width={'100%'} />
      </div>
    )
  }

  return null;
}
