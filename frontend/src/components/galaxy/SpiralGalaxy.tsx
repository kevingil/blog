import { useEffect, useRef } from 'react';
import type { ThemeMode } from './config';
import { mountGalaxy, type GalaxyHandle } from './mount';

export interface SpiralGalaxyProps {
  /** Light mode draws dark particles on a near-white sky. */
  theme: ThemeMode;
  zIndex?: number;
}

/** Full-viewport animated spiral galaxy, fixed behind the page content. */
export function SpiralGalaxy({ theme, zIndex = -1 }: SpiralGalaxyProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const galaxyRef = useRef<GalaxyHandle | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const galaxy = mountGalaxy(canvas);
    galaxyRef.current = galaxy;
    return () => {
      galaxy.destroy();
      galaxyRef.current = null;
    };
  }, []);

  // Declared after the mount effect so it also applies the initial theme
  // before the first frame is drawn.
  useEffect(() => {
    galaxyRef.current?.setTheme(theme);
  }, [theme]);

  return (
    <div className="pointer-events-none fixed inset-0 overflow-hidden" style={{ zIndex }} aria-hidden="true">
      <canvas ref={canvasRef} className="pointer-events-auto h-full w-full" />
    </div>
  );
}
