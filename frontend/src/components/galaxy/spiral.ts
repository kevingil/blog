import { GALAXY } from './config';
import { uniform, type Random } from './random';

const TWO_PI = 2 * Math.PI;

/** One family of logarithmic spiral arms, evenly spaced around the disk. */
interface ArmFamily {
  armCount: number;
  /** Angle of the first arm's root. */
  startAngle: number;
  /** Growth rate b in r = a·e^(bθ); smaller winds tighter. */
  tightness: number;
  /** Root radius a, as a multiple of the core radius. */
  rootRadius: number;
  /** Only contributes beyond this radius (as a multiple of the core radius). */
  minRadius?: number;
  amplitude: number;
}

/** Arm widths (Gaussian sigma, radians) differ per population. */
export interface ArmWidths {
  main: number;
  inner: number;
  outer: number;
  /** Spur width is `min + random() * span`. */
  spur: { min: number; span: number };
}

/** Dust traces broad lanes; stars hug the arms more tightly. */
export const DUST_ARM_WIDTHS: ArmWidths = {
  main: 0.4,
  inner: 0.3,
  outer: 0.5,
  spur: { min: 0.2, span: 0.15 },
};

export const STAR_ARM_WIDTHS: ArmWidths = {
  main: 0.3,
  inner: 0.25,
  outer: 0.4,
  spur: { min: 0.15, span: 0.1 },
};

const MAIN_ARMS: ArmFamily = { armCount: 2, startAngle: 0, tightness: 0.25, rootRadius: 0.8, amplitude: 5 };
const INNER_ARMS: ArmFamily = { armCount: 2, startAngle: Math.PI / 2, tightness: 0.18, rootRadius: 0.4, amplitude: 3 };
const OUTER_ARMS: ArmFamily = {
  armCount: 2,
  startAngle: Math.PI / 4,
  tightness: 0.35,
  rootRadius: 1.2,
  minRadius: 1.5,
  amplitude: 2,
};

/** Short, randomly shaped arm fragments that break up the clean spiral. */
const SPUR_COUNT = 12;

/** Angle of main arm `arm` (0 or 1) at radius r, along its centre line. */
export function mainArmAngle(r: number, arm: number): number {
  const rootRadius = GALAXY.coreRadius * MAIN_ARMS.rootRadius;
  const rootAngle = (arm * TWO_PI) / MAIN_ARMS.armCount + MAIN_ARMS.startAngle;
  return -(1 / MAIN_ARMS.tightness) * Math.log(Math.max(r, rootRadius) / rootRadius) + rootAngle;
}

export const MAIN_ARM_COUNT = MAIN_ARMS.armCount;

/** Wraps an angle into [-π, π). */
export function wrapAngle(angle: number): number {
  return (((angle % TWO_PI) + 3 * Math.PI) % TWO_PI) - Math.PI;
}

/**
 * Strength of a single logarithmic arm at polar position (r, θ): a Gaussian of
 * the angular distance from the arm's centre line.
 */
function armStrength(
  r: number,
  theta: number,
  rootAngle: number,
  tightness: number,
  rootRadius: number,
  width: number,
  amplitude: number,
): number {
  const armAngle = -(1 / tightness) * Math.log(Math.max(r, rootRadius) / rootRadius) + rootAngle;
  const distance = wrapAngle(theta - armAngle);
  return Math.exp(-Math.pow(distance / width, 2)) * amplitude;
}

/**
 * How strongly position (r, θ) sits on spiral structure, fading out past the
 * disk edge. Dust takes the strongest single arm (`max`); stars accumulate
 * every arm they overlap (`sum`), which makes crossings brighter.
 */
export function spiralStrength(
  r: number,
  theta: number,
  widths: ArmWidths,
  combine: 'max' | 'sum',
  random: Random,
): number {
  const { coreRadius, diskRadius } = GALAXY;
  let strength = 0;
  const add = (value: number) => {
    strength = combine === 'max' ? Math.max(strength, value) : strength + value;
  };

  const families: Array<[ArmFamily, number]> = [
    [MAIN_ARMS, widths.main],
    [INNER_ARMS, widths.inner],
    [OUTER_ARMS, widths.outer],
  ];
  for (const [family, width] of families) {
    if (family.minRadius !== undefined && r <= coreRadius * family.minRadius) continue;
    for (let arm = 0; arm < family.armCount; arm++) {
      const rootAngle = (arm * TWO_PI) / family.armCount + family.startAngle;
      add(armStrength(r, theta, rootAngle, family.tightness, coreRadius * family.rootRadius, width, family.amplitude));
    }
  }

  for (let spur = 0; spur < SPUR_COUNT; spur++) {
    const rootAngle = (spur / SPUR_COUNT) * TWO_PI;
    const tightness = uniform(random, 0.15, 0.25);
    const rootRadius = coreRadius * uniform(random, 0.3, 0.6);
    const width = uniform(random, widths.spur.min, widths.spur.span);
    const amplitude = uniform(random, 1, 1);
    add(armStrength(r, theta, rootAngle, tightness, rootRadius, width, amplitude));
  }

  return strength * Math.exp(-Math.pow(r / (diskRadius * 1.1), 1.8));
}
