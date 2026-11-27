import { ANIMATION, DENSITY_WAVE, GALAXY } from './config';
import { ARMS, MAIN_ARM_COUNT, wrapAngle } from './spiral';

const TWO_PI = 2 * Math.PI;

/**
 * Angular speed of the spiral pattern. It matches the material's speed at
 * corotation (beyond the disk), so everything inside overtakes the arms and
 * flows through them from their inner, upstream edge.
 */
export const PATTERN_SPEED =
  (GALAXY.coreAngularSpeed * GALAXY.coreRadius) / (GALAXY.diskRadius * DENSITY_WAVE.corotation);

/** Something orbiting through the pattern. */
export interface Orbiter {
  /** Initial angle, which is also its pattern-frame angle at time 0. */
  angle: number;
  angularSpeed: number;
}

/** Where an orbiter is in the pattern's frame at simulation `time`. */
export function patternAngle(orbiter: Orbiter, time: number): number {
  return orbiter.angle + (orbiter.angularSpeed - PATTERN_SPEED) * time * ANIMATION.rotationSpeed;
}

/**
 * How strongly each arm lights gas, relative to the main pair. Kept
 * low for the secondary families so the two main arms read as the spiral
 * instead of a set of rings.
 */
const ARM_GLOW = ARMS.map((_, i) => (i < MAIN_ARM_COUNT ? 1 : 0.35));

/** 0–1: how much an orbiter at pattern angle `phase` is lit by being in an arm. */
export function armGlow(arms: readonly number[], phase: number): number {
  let lit = 0;
  for (let i = 0; i < arms.length; i++) {
    if (Number.isNaN(arms[i])) continue;
    const { width } = ARMS[i].family;
    const strength = ARM_GLOW[i] * Math.exp(-Math.pow(wrapAngle(phase - arms[i]) / width, 2));
    lit = Math.max(lit, strength);
  }
  return DENSITY_WAVE.interArmGlow + (1 - DENSITY_WAVE.interArmGlow) * lit;
}

/** 0–1: how deep in a dust lane (on a main arm's upstream edge) the orbiter is. */
export function laneDepth(arms: readonly number[], phase: number): number {
  let depth = 0;
  for (let i = 0; i < MAIN_ARM_COUNT; i++) {
    const offset = wrapAngle(phase - (arms[i] + DENSITY_WAVE.laneOffset));
    depth = Math.max(depth, Math.exp(-Math.pow(offset / DENSITY_WAVE.laneWidth, 2)));
  }
  return depth;
}

/**
 * 0–1: how bright a young star is, given how long ago it crossed a main arm.
 * It rises quickly at birth and fades as it drifts downstream; stars
 * approaching an arm are not yet born.
 */
export function youth(arms: readonly number[], phase: number): number {
  let brightest = 0;
  for (let i = 0; i < MAIN_ARM_COUNT; i++) {
    const sinceBirth = (((phase - (arms[i] + DENSITY_WAVE.birthOffset)) % TWO_PI) + TWO_PI) % TWO_PI;
    const brightness = (1 - Math.exp(-sinceBirth / 0.05)) * Math.exp(-sinceBirth / DENSITY_WAVE.youthDecay);
    brightest = Math.max(brightest, brightness);
  }
  return brightest;
}
