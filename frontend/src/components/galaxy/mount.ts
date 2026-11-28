import { ANIMATION, layoutForWidth, MAX_PIXEL_RATIO, POINTER_TILT, type ThemeMode } from './config';
import { FLOATS_PER_VERTEX, writeFrame, type FrameState } from './frame';
import { generateParticles } from './particles';
import type { Random } from './random';
import { GalaxyRenderer } from './renderer';

export interface GalaxyOptions {
  theme?: ThemeMode;
  /** Random source for the particle layout and light-mode shimmer. Defaults to `Math.random`. */
  random?: Random;
}

export interface GalaxyHandle {
  setTheme(theme: ThemeMode): void;
  /** Stops the animation, removes listeners and frees GPU resources. */
  destroy(): void;
}

/**
 * Starts the galaxy animation on `canvas`, sized to the window. It follows
 * window resizes and tilts slightly toward the mouse. Framework-agnostic; see
 * `SpiralGalaxy` for the React wrapper.
 */
export function mountGalaxy(canvas: HTMLCanvasElement, options: GalaxyOptions = {}): GalaxyHandle {
  const random = options.random ?? Math.random;
  const particles = generateParticles(random);
  const vertices = new Float32Array(particles.length * FLOATS_PER_VERTEX);

  let renderer = GalaxyRenderer.create(canvas, particles.length);
  let frameId: number | null = null;
  let lastFrameAt: number | null = null;
  let pixelRatio = 1;
  const targetTilt = { pitch: 0, yaw: 0 };
  const state: FrameState = {
    time: 0,
    pitch: 0,
    yaw: 0,
    layout: layoutForWidth(window.innerWidth),
    width: 0,
    height: 0,
    theme: options.theme ?? 'dark',
  };

  const resize = () => {
    // Render at device resolution so star cores stay crisp; layout stays in CSS px.
    pixelRatio = Math.min(window.devicePixelRatio || 1, MAX_PIXEL_RATIO);
    state.width = window.innerWidth;
    state.height = window.innerHeight;
    state.layout = layoutForWidth(window.innerWidth);
    canvas.width = Math.round(state.width * pixelRatio);
    canvas.height = Math.round(state.height * pixelRatio);
  };

  const onMouseMove = (event: MouseEvent) => {
    // -1…1 from the centre of the viewport.
    const nx = (event.clientX - window.innerWidth / 2) / (window.innerWidth / 2);
    const ny = (event.clientY - window.innerHeight / 2) / (window.innerHeight / 2);
    targetTilt.pitch = ny * POINTER_TILT.maxAngle;
    targetTilt.yaw = nx * POINTER_TILT.maxAngle;
  };

  const onMouseLeave = () => {
    targetTilt.pitch = 0;
    targetTilt.yaw = 0;
  };

  const frame = (now: number) => {
    if (!renderer) return;
    state.time += (now - (lastFrameAt ?? now)) * ANIMATION.timePerMs;
    lastFrameAt = now;
    state.pitch += (targetTilt.pitch - state.pitch) * POINTER_TILT.easing;
    state.yaw += (targetTilt.yaw - state.yaw) * POINTER_TILT.easing;

    const counts = writeFrame(particles, state, vertices, random);
    renderer.draw(vertices, counts, {
      theme: state.theme,
      cssWidth: state.width,
      cssHeight: state.height,
      pixelRatio,
    });
    frameId = requestAnimationFrame(frame);
  };

  const start = () => {
    if (renderer && frameId === null) frameId = requestAnimationFrame(frame);
  };

  const stop = () => {
    if (frameId !== null) cancelAnimationFrame(frameId);
    frameId = null;
    // Resume without a jump in time.
    lastFrameAt = null;
  };

  const onContextLost = (event: Event) => {
    event.preventDefault(); // allows the browser to restore the context
    stop();
    renderer = null;
  };

  const onContextRestored = () => {
    renderer = GalaxyRenderer.create(canvas, particles.length);
    start();
  };

  resize();
  window.addEventListener('resize', resize);
  window.addEventListener('mousemove', onMouseMove);
  document.documentElement.addEventListener('mouseleave', onMouseLeave);
  canvas.addEventListener('webglcontextlost', onContextLost);
  canvas.addEventListener('webglcontextrestored', onContextRestored);
  start();

  return {
    setTheme(theme) {
      state.theme = theme;
    },
    destroy() {
      stop();
      window.removeEventListener('resize', resize);
      window.removeEventListener('mousemove', onMouseMove);
      document.documentElement.removeEventListener('mouseleave', onMouseLeave);
      canvas.removeEventListener('webglcontextlost', onContextLost);
      canvas.removeEventListener('webglcontextrestored', onContextRestored);
      renderer?.dispose();
      renderer = null;
    },
  };
}
