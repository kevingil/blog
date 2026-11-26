/**
 * Animated spiral galaxy backdrop (WebGL, no dependencies).
 *
 * - `SpiralGalaxy`: React component for the public site.
 * - `mountGalaxy`: the same animation on any canvas, without React.
 *
 * Layout: `config` holds every tunable constant; `particles` (with `spiral`
 * and `palette`) builds the scene once; `frame` projects it each frame into a
 * vertex buffer; `renderer` and `shaders` draw it; `mount` runs the loop.
 * Nothing here imports from the rest of the app.
 */
export { SpiralGalaxy, type SpiralGalaxyProps } from './SpiralGalaxy';
export { mountGalaxy, type GalaxyHandle, type GalaxyOptions } from './mount';
export type { ThemeMode } from './config';
export { seededRandom, type Random } from './random';
