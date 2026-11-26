import { ANIMATION, CAMERA, GALAXY, NEBULA, STAR_HALO, type GalaxyLayout, type ThemeMode } from './config';
import { toLightMode, type Rgb } from './palette';
import type { Particle } from './particles';
import type { Random } from './random';

/** Interleaved vertex layout: x, y, size, r, g, b, alpha, kind, extra. */
export const FLOATS_PER_VERTEX = 9;

/**
 * How the shader draws each vertex. `extra` carries the halo ratio for stars
 * and the texture seed for smoke. Mirrored as `#define`s in the shader.
 */
export const SPRITE = {
  star: 0,
  armStar: 1,
  nucleus: 2,
  gas: 3,
  dust: 4,
  smoke: 5,
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
}

/** The galaxy disk's outline on screen, for shaping smoke sprites. */
export interface DiskEllipse {
  /** Angle of the major axis, clockwise from +x in screen space (y down). */
  angle: number;
  /** Minor / major axis ratio, floored at `NEBULA.minAspect`. */
  aspect: number;
}

const COS_INCLINATION = Math.cos(-GALAXY.inclination);
const SIN_INCLINATION = Math.sin(-GALAXY.inclination);
const COS_ORIENTATION = Math.cos(GALAXY.orientation);
const SIN_ORIENTATION = Math.sin(GALAXY.orientation);

/**
 * Turns a point in the galaxy's frame into screen-aligned axes (before the
 * horizontal mirror, zoom and offset): pointer tilt, then the fixed
 * inclination and position angle. Returns [x, y, depth].
 */
function orient(x: number, y: number, z: number, state: FrameState): [number, number, number] {
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
  return [
    yawedX * COS_ORIENTATION - inclinedY * SIN_ORIENTATION,
    yawedX * SIN_ORIENTATION + inclinedY * COS_ORIENTATION,
    depth,
  ];
}

/** Places a particle on screen, applying differential rotation and perspective. */
export function projectParticle(particle: Particle, state: FrameState): ScreenPoint {
  const { time, layout, width } = state;
  const angle = particle.angle + particle.angularSpeed * time * ANIMATION.rotationSpeed;
  let x = particle.radius * Math.cos(angle);
  let y = particle.radius * Math.sin(angle);

  if (particle.kind === 'bulge') {
    // Slow turbulent drift of the central gas.
    x += Math.sin(time * 0.00005 + particle.x * 0.01) * 0.5;
    y += Math.cos(time * 0.00005 + particle.y * 0.01) * 0.5;
  }

  const [screenX, screenY, depth] = orient(x, y, particle.z, state);
  const scale = (CAMERA.distance / (CAMERA.distance + depth)) * layout.zoom;
  return {
    // The galaxy is mirrored horizontally on screen.
    x: width / 2 - screenX * scale + layout.offsetX,
    y: layout.offsetY + screenY * scale,
    scale,
  };
}

/**
 * The on-screen ellipse of a circle in the disk plane: the singular vectors
 * of the 2×2 map from disk axes to screen axes.
 */
export function diskEllipse(state: FrameState): DiskEllipse {
  const [ux, uy] = orient(1, 0, 0, state);
  const [vx, vy] = orient(0, 1, 0, state);
  // Screen x is mirrored.
  const a = [-ux, uy];
  const b = [-vx, vy];
  const sxx = a[0] * a[0] + b[0] * b[0];
  const syy = a[1] * a[1] + b[1] * b[1];
  const sxy = a[0] * a[1] + b[0] * b[1];
  const mean = (sxx + syy) / 2;
  const spread = Math.sqrt(((sxx - syy) / 2) ** 2 + sxy * sxy);
  const major = Math.sqrt(mean + spread);
  const minor = Math.sqrt(Math.max(0, mean - spread));
  return {
    angle: 0.5 * Math.atan2(2 * sxy, sxx - syy),
    aspect: Math.max(NEBULA.minAspect, minor / major),
  };
}

function isOnScreen({ x, y }: ScreenPoint, radius: number, width: number, height: number): boolean {
  const margin = CAMERA.cullMargin + radius;
  return x >= -margin && x <= width + margin && y >= -margin && y <= height + margin;
}

const clamp01 = (value: number) => Math.max(0, Math.min(1, value));

function spriteFor(particle: Particle): number {
  switch (particle.kind) {
    case 'nebula':
      return SPRITE.smoke;
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
 * Writes one vertex per visible particle into `out`, in particle order, and
 * returns how many were written. `random` drives light mode's shimmer.
 */
export function writeFrame(
  particles: readonly Particle[],
  state: FrameState,
  out: Float32Array,
  random: Random = Math.random,
): number {
  const { time, theme, width, height } = state;
  let count = 0;

  for (const particle of particles) {
    const point = projectParticle(particle, state);
    const sprite = spriteFor(particle);
    const isStar = sprite === SPRITE.star || sprite === SPRITE.armStar || sprite === SPRITE.nucleus;

    let size = particle.size * point.scale;
    let alpha = particle.brightness * Math.min(1, point.scale);
    let extra = particle.seed;

    const [baseR, baseG, baseB] = particle.color;
    let color: Rgb = [baseR / 255, baseG / 255, baseB / 255];

    if (sprite === SPRITE.gas) {
      // The central gas breathes: size, hue and opacity drift slowly.
      size *= 1.5 + Math.sin(time * 0.0001 + particle.x * 0.01) * 0.3;
      const shift = Math.sin(time * 0.00005 + particle.y * 0.01) * 0.1;
      color = [clamp01(color[0] + shift), clamp01(color[1] + shift * 0.5), clamp01(color[2] + shift * 0.8)];
      alpha *= 0.8 + 0.2 * Math.sin(time * 0.00008 + particle.x * 0.005 + particle.y * 0.003);
    }
    if (theme === 'light') {
      color = toLightMode(particle.color, color, random);
      alpha = Math.min(1.0, alpha * 2.0);
    }

    size = Math.max(1.0, size);
    if (isStar) {
      // The sprite holds the halo; the core keeps the star's own size.
      extra = STAR_HALO.min + STAR_HALO.perBrightness * clamp01(alpha);
      size *= extra;
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
    out[offset + 8] = extra;
    count++;
  }

  return count;
}
