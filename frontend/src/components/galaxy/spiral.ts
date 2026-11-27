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
  /** Only present beyond this radius (as a multiple of the core radius). */
  minRadius?: number;
  amplitude: number;
  /** Gaussian half-width of the arm, in radians. */
  width: number;
}

/**
 * The spiral pattern, strongest first: two dominant arms, a tighter inner
 * pair, and a looser outer pair. All are trailing spirals for rotation toward
 * increasing angle.
 */
export const ARM_FAMILIES: readonly ArmFamily[] = [
  { armCount: 2, startAngle: 0, tightness: 0.25, rootRadius: 0.8, amplitude: 5, width: 0.4 },
  { armCount: 2, startAngle: Math.PI / 2, tightness: 0.18, rootRadius: 0.4, amplitude: 3, width: 0.3 },
  { armCount: 2, startAngle: Math.PI / 4, tightness: 0.35, rootRadius: 1.2, minRadius: 1.5, amplitude: 2, width: 0.5 },
];

/** Every arm of every family, flattened in the order `armAngles` reports them. */
export const ARMS = ARM_FAMILIES.flatMap((family) =>
  Array.from({ length: family.armCount }, (_, arm) => ({ family, arm })),
);

/** The two dominant arms are the first entries of `ARMS`. */
export const MAIN_ARM_COUNT = ARM_FAMILIES[0].armCount;

/** Short, randomly shaped arm fragments that break up the clean spiral. */
const SPUR_COUNT = 12;
const SPUR_WIDTH = { min: 0.2, span: 0.15 };

/** Wraps an angle into [-π, π). */
export function wrapAngle(angle: number): number {
  return (((angle % TWO_PI) + 3 * Math.PI) % TWO_PI) - Math.PI;
}

function logSpiralAngle(r: number, rootAngle: number, tightness: number, rootRadius: number): number {
  return -(1 / tightness) * Math.log(Math.max(r, rootRadius) / rootRadius) + rootAngle;
}

/**
 * Centre-line angle of each arm in `ARMS` at radius r, in the pattern's
 * frame, or NaN where that arm does not reach.
 */
export function armAngles(r: number): number[] {
  return ARMS.map(({ family, arm }) => {
    if (family.minRadius !== undefined && r <= GALAXY.coreRadius * family.minRadius) return NaN;
    const rootAngle = (arm * TWO_PI) / family.armCount + family.startAngle;
    return logSpiralAngle(r, rootAngle, family.tightness, GALAXY.coreRadius * family.rootRadius);
  });
}

/**
 * How strongly position (r, θ) sits on spiral structure (strongest arm or
 * spur), fading out past the disk edge. Spurs are random, so this is for
 * one-off placement, not per-frame use.
 */
export function spiralStrength(r: number, theta: number, random: Random): number {
  let strength = 0;
  armAngles(r).forEach((angle, i) => {
    if (Number.isNaN(angle)) return;
    const { amplitude, width } = ARMS[i].family;
    strength = Math.max(strength, amplitude * Math.exp(-Math.pow(wrapAngle(theta - angle) / width, 2)));
  });

  for (let spur = 0; spur < SPUR_COUNT; spur++) {
    const rootAngle = (spur / SPUR_COUNT) * TWO_PI;
    const tightness = uniform(random, 0.15, 0.25);
    const rootRadius = GALAXY.coreRadius * uniform(random, 0.3, 0.6);
    const width = uniform(random, SPUR_WIDTH.min, SPUR_WIDTH.span);
    const amplitude = uniform(random, 1, 1);
    const angle = logSpiralAngle(r, rootAngle, tightness, rootRadius);
    strength = Math.max(strength, amplitude * Math.exp(-Math.pow(wrapAngle(theta - angle) / width, 2)));
  }

  return strength * Math.exp(-Math.pow(r / (GALAXY.diskRadius * 1.1), 1.8));
}
