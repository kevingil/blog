import { decode } from 'blurhash';
import { useEffect, useRef, useState } from 'react';

type BlurhashImageProps = {
  src: string;
  alt?: string;
  blurhash?: string | null;
  className?: string;
  imgClassName?: string;
};

const HOLD_MS = 900;

export function BlurhashImage({
  src,
  alt = '',
  blurhash,
  className,
  imgClassName,
}: BlurhashImageProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [revealed, setRevealed] = useState(!blurhash);

  useEffect(() => {
    setRevealed(!blurhash);
  }, [src, blurhash]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !blurhash) return;
    const width = 32;
    const height = 32;
    try {
      const pixels = decode(blurhash, width, height);
      canvas.width = width;
      canvas.height = height;
      const context = canvas.getContext('2d');
      if (!context) return;
      const imageData = context.createImageData(width, height);
      imageData.data.set(pixels);
      context.putImageData(imageData, 0, 0);
    } catch {
      canvas.width = 0;
      canvas.height = 0;
    }
  }, [blurhash]);

  return (
    <div className={className ? `relative overflow-hidden ${className}` : 'relative overflow-hidden'}>
      {blurhash ? (
        <canvas
          ref={canvasRef}
          aria-hidden
          className="absolute inset-0 h-full w-full"
          style={{
            opacity: revealed ? 0 : 1,
            transition: 'opacity 700ms ease',
          }}
        />
      ) : null}
      <img
        src={src}
        alt={alt}
        className={imgClassName ?? 'h-full w-full object-cover'}
        style={{
          opacity: blurhash && !revealed ? 0 : 1,
          transition: 'opacity 700ms ease',
        }}
        onLoad={() => {
          if (!blurhash) {
            setRevealed(true);
            return;
          }
          window.setTimeout(() => setRevealed(true), HOLD_MS);
        }}
      />
    </div>
  );
}
