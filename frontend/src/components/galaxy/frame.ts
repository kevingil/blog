import { ANIMATION, CAMERA, GALAXY, type GalaxyLayout, type ThemeMode } from './config';
import { toLightMode, type Rgb } from './palette';
import type { Particle } from './particles';
import type { Random } from './random';

/** Interleaved vertex layout: x, y, size, r, g, b, alpha, spiralArm. */
export const FLOATS_PER_VERTEX = 8;

export interface FrameState {
  /** Simulation time; see `ANIMATION.timePerMs`. */
  time: number;
  /** Pointer-driven pitch (about the x axis), in radians. */
  pitch: number;
  /** Pointer-driven yaw (about the view axis), in radians. */
  yaw: number;
  layout: GalaxyLayout;
  /** Canvas size in device pixels. */
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

const COS_INCLINATION = Math.cos(-GALAXY.inclination);
const SIN_INCLINATION = Math.sin(-GALAXY.inclination);
const COS_ORIENTATION = Math.cos(GALAXY.orientation);
const SIN_ORIENTATION = Math.sin(GALAXY.orientation);

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
  };
}

function isOnScreen({ x, y }: ScreenPoint, width: number, height: number): boolean {
  const margin = CAMERA.cullMargin;
  return x >= -margin && x <= width + margin && y >= -margin && y <= height + margin;
}

const clamp01 = (value: number) => Math.max(0, Math.min(1, value));

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
    if (!isOnScreen(point, width, height)) continue;

    const isBulge = particle.kind === 'bulge';
    let size = particle.size * point.scale;
    let alpha = particle.brightness * Math.min(1, point.scale);

    const [baseR, baseG, baseB] = particle.color;
    let color: Rgb = [baseR / 255, baseG / 255, baseB / 255];

    if (isBulge) {
      // The central gas breathes: size, hue and opacity drift slowly.
      size *= 1.5 + Math.sin(time * 0.0001 + particle.x * 0.01) * 0.3;
      const shift = Math.sin(time * 0.00005 + particle.y * 0.01) * 0.1;
      color = [clamp01(color[0] + shift), clamp01(color[1] + shift * 0.5), clamp01(color[2] + shift * 0.8)];
    }
    if (theme === 'light') {
      color = toLightMode(particle.color, color, random);
      alpha = Math.min(1.0, alpha * 2.0);
    }
    if (isBulge) {
      alpha *= 0.8 + 0.2 * Math.sin(time * 0.00008 + particle.x * 0.005 + particle.y * 0.003);
    }

    const offset = count * FLOATS_PER_VERTEX;
    out[offset] = point.x;
    out[offset + 1] = point.y;
    out[offset + 2] = Math.max(1.0, size);
    out[offset + 3] = color[0];
    out[offset + 4] = color[1];
    out[offset + 5] = color[2];
    out[offset + 6] = alpha;
    out[offset + 7] = particle.spiralArm ? 1.0 : 0.0;
    count++;
  }

  return count;
}
