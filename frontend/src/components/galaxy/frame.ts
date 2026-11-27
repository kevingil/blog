import { ANIMATION, CAMERA, GALAXY, STAR_HALO, type GalaxyLayout, type ThemeMode } from './config';
import { toLightMode, type Rgb } from './palette';
import type { Particle } from './particles';
import type { Random } from './random';
import { armGlow, laneDepth, patternAngle, youth } from './wave';

/** Interleaved vertex layout: x, y, size, r, g, b, alpha, kind, halo. */
export const FLOATS_PER_VERTEX = 9;

/** How the shader draws each vertex. Mirrored as `#define`s in the shader. */
export const SPRITE = {
  star: 0,
  armStar: 1,
  nucleus: 2,
  gas: 3,
  dust: 4,
  cloud: 5,
  dustLane: 6,
} as const;

export interface FrameState {
  /** Simulation time; see `ANIMATION.timePerMs`. */
  time: number;
  /** Pointer-driven pitch (about the x axis), in radians. */
  pitch: number;
  /** Pointer-driven yaw (about the view axis), in radians. */
  yaw: number;
  layout: GalaxyLayout;
  /** Viewport size in CSS pixels. */
  width: number;
  height: number;
  theme: ThemeMode;
}

export interface ScreenPoint {
  x: number;
  y: number;
  /** Perspective scale, including zoom. */
  scale: number;
  /** Distance behind the galaxy's centre plane along the view axis; negative is nearer. */
  depth: number;
}

/**
 * Vertex counts for one frame, in draw order: soft gas first (drawn into a
 * low-resolution layer), then the sharp emissive sprites, then absorbing
 * dust lanes.
 */
export interface FrameCounts {
  total: number;
  /** Leading vertices that are soft gas. */
  soft: number;
  /** Leading vertices that emit light (soft gas included). */
  emissive: number;
}

const COS_INCLINATION = Math.cos(-GALAXY.inclination);
const SIN_INCLINATION = Math.sin(-GALAXY.inclination);
const COS_ORIENTATION = Math.cos(GALAXY.orientation);
const SIN_ORIENTATION = Math.sin(GALAXY.orientation);

/** Neutral ink for gas in light mode, where colour would look garish. */
const LIGHT_MODE_GAS_INK: Rgb = [0.24, 0.22, 0.22];
/** Dust lanes in light mode: a slightly warmer, darker ink. */
const LIGHT_MODE_DUST_INK: Rgb = [0.26, 0.18, 0.14];

/**
 * Places a particle on screen: differential rotation in the galaxy plane,
 * pointer tilt, the fixed inclination and position angle, then perspective.
 */
export function projectParticle(particle: Particle, state: FrameState): ScreenPoint {
  const { time, layout, width } = state;
  const angle = particle.angle + particle.angularSpeed * time * ANIMATION.rotationSpeed;
  let x = particle.radius * Math.cos(angle);
  let y = particle.radius * Math.sin(angle);
  const z = particle.z;

  if (particle.kind === 'bulge') {
    // Slow turbulent drift of the central gas.
    x += Math.sin(time * 0.00005 + particle.x * 0.01) * 0.5;
    y += Math.cos(time * 0.00005 + particle.y * 0.01) * 0.5;
  }

  // Pointer pitch, about the x axis.
  const cosPitch = Math.cos(state.pitch);
  const sinPitch = Math.sin(state.pitch);
  const pitchedY = y * cosPitch - z * sinPitch;
  const pitchedZ = y * sinPitch + z * cosPitch;

  // Pointer yaw, about the z axis.
  const cosYaw = Math.cos(state.yaw);
  const sinYaw = Math.sin(state.yaw);
  const yawedX = x * cosYaw - pitchedY * sinYaw;
  const yawedY = x * sinYaw + pitchedY * cosYaw;

  // Fixed inclination, then position angle on screen.
  const inclinedY = yawedY * COS_INCLINATION - pitchedZ * SIN_INCLINATION;
  const depth = yawedY * SIN_INCLINATION + pitchedZ * COS_INCLINATION;
  const screenX = yawedX * COS_ORIENTATION - inclinedY * SIN_ORIENTATION;
  const screenY = yawedX * SIN_ORIENTATION + inclinedY * COS_ORIENTATION;

  const scale = (CAMERA.distance / (CAMERA.distance + depth)) * layout.zoom;
  return {
    // The galaxy is mirrored horizontally on screen.
    x: width / 2 - screenX * scale + layout.offsetX,
    y: layout.offsetY + screenY * scale,
    scale,
    depth,
  };
}

function isOnScreen({ x, y }: ScreenPoint, radius: number, width: number, height: number): boolean {
  const margin = CAMERA.cullMargin + radius;
  return x >= -margin && x <= width + margin && y >= -margin && y <= height + margin;
}

const clamp01 = (value: number) => Math.max(0, Math.min(1, value));

function spriteFor(particle: Particle): number {
  switch (particle.kind) {
    case 'haze':
    case 'diskLight':
    case 'cloud':
    case 'hii':
    case 'satellite':
      return SPRITE.cloud;
    case 'dustLane':
      return SPRITE.dustLane;
    case 'bulge':
      return SPRITE.gas;
    case 'dust':
      return SPRITE.dust;
    case 'nucleus':
      return SPRITE.nucleus;
    default:
      return particle.spiralArm ? SPRITE.armStar : SPRITE.star;
  }
}

/**
 * Dust only hides what is behind it. Lanes on the near side of the disk
 * (in front of the bulge) keep full opacity; far-side lanes mostly fade, since
 * the light they would block is in front of them.
 */
function dustLaneVisibility(depth: number): number {
  return Math.max(0.15, Math.min(1, 0.5 - depth / (GALAXY.diskRadius * 0.5)));
}

/** Density-wave multiplier on a particle's brightness: 1 for particles it does not affect. */
function waveFactor(particle: Particle, time: number): number {
  if (!particle.arms) return 1;
  const phase = patternAngle(particle, time);
  if (particle.kind === 'cloud') return armGlow(particle.arms, phase);
  if (particle.kind === 'dustLane') return laneDepth(particle.arms, phase);
  // Young stars and the nebulae around them share a lifetime.
  return youth(particle.arms, phase);
}

/**
 * Gaussian gas and dust sprites show about half their width, so they are
 * drawn this much larger to keep their apparent size.
 */
const SOFT_SPRITE_SCALE = 1.8;

/** Below this opacity a sprite contributes nothing visible, so it is skipped. */
const MIN_ALPHA = 0.003;

/**
 * Writes one vertex per visible particle into `out`, in particle order, and
 * returns the counts. Particles must list absorbing dust lanes last (as
 * `generateParticles` does). `random` drives light mode's star shimmer.
 */
export function writeFrame(
  particles: readonly Particle[],
  state: FrameState,
  out: Float32Array,
  random: Random = Math.random,
): FrameCounts {
  const { time, theme, width, height } = state;
  const light = theme === 'light';
  let count = 0;
  let soft = 0;
  let emissive = 0;

  for (const particle of particles) {
    // Cheap test first: most dust and young stars are dark at any moment.
    const wave = waveFactor(particle, time);
    if (particle.brightness * wave < MIN_ALPHA) continue;

    const point = projectParticle(particle, state);
    const sprite = spriteFor(particle);
    const isStar = sprite === SPRITE.star || sprite === SPRITE.armStar || sprite === SPRITE.nucleus;
    const isGasOrDust = sprite === SPRITE.cloud || sprite === SPRITE.dustLane;

    let size = particle.size * point.scale;
    let alpha = particle.brightness * wave * Math.min(1, point.scale);
    let halo = 0;

    const [baseR, baseG, baseB] = particle.color;
    let color: Rgb = [baseR / 255, baseG / 255, baseB / 255];

    if (sprite === SPRITE.gas) {
      // The central gas breathes: size, hue and opacity drift slowly.
      size *= 1.5 + Math.sin(time * 0.0001 + particle.x * 0.01) * 0.3;
      const shift = Math.sin(time * 0.00005 + particle.y * 0.01) * 0.1;
      color = [clamp01(color[0] + shift), clamp01(color[1] + shift * 0.5), clamp01(color[2] + shift * 0.8)];
      alpha *= 0.8 + 0.2 * Math.sin(time * 0.00008 + particle.x * 0.005 + particle.y * 0.003);
    }
    if (sprite === SPRITE.dustLane) {
      alpha *= dustLaneVisibility(point.depth);
    }
    if (light && isGasOrDust) {
      // A faint wash: ink pools far more visibly than light adds up.
      color = sprite === SPRITE.dustLane ? LIGHT_MODE_DUST_INK : LIGHT_MODE_GAS_INK;
      alpha *= sprite === SPRITE.dustLane ? 0.6 : 0.35;
    } else if (light) {
      color = toLightMode(particle.color, color, random);
      alpha = Math.min(1.0, alpha * 2.0);
    }

    if (sprite === SPRITE.gas || sprite === SPRITE.dust) size *= SOFT_SPRITE_SCALE;
    size = Math.max(1.0, size);
    if (isStar) {
      // The sprite holds the halo; the core keeps the star's own size.
      halo = STAR_HALO.min + STAR_HALO.perBrightness * clamp01(alpha);
      size *= halo;
    }
    if (!isOnScreen(point, size / 2, width, height)) continue;

    const offset = count * FLOATS_PER_VERTEX;
    out[offset] = point.x;
    out[offset + 1] = point.y;
    out[offset + 2] = size;
    out[offset + 3] = color[0];
    out[offset + 4] = color[1];
    out[offset + 5] = color[2];
    out[offset + 6] = alpha;
    out[offset + 7] = sprite;
    out[offset + 8] = halo;
    count++;
    // Particles list all soft gas first, so it forms a prefix.
    if (sprite === SPRITE.cloud && soft === count - 1) soft = count;
    if (sprite !== SPRITE.dustLane) emissive = count;
  }

  return { total: count, soft, emissive };
}
