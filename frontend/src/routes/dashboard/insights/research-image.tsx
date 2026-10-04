import { useState } from "react";

interface ResearchImageProps {
  src?: string;
  className: string;
}

export function ResearchImage({ src, className }: ResearchImageProps) {
  const [hidden, setHidden] = useState(false);
  if (!src || hidden) {
    return null;
  }
  return (
    <img
      src={src}
      alt=""
      className={className}
      loading="lazy"
      referrerPolicy="no-referrer"
      onError={() => setHidden(true)}
    />
  );
}
