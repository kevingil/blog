import { GALAXY } from './config';
import { uniformInt, type Random } from './random';

/** An sRGB colour with 0–255 channels. */
export type Rgb255 = readonly [number, number, number];
/** An sRGB colour with 0–1 channels, as the shader consumes it. */
export type Rgb = [number, number, number];

/**
 * Approximate blackbody colour for a stellar temperature in kelvin
 * (3 000 K is red, 30 000 K blue-white). Returns 0–1 channels.
 */
export function temperatureToRgb(kelvin: number): Rgb {
  const t = Math.max(1000, Math.min(40000, kelvin));
  if (t < 3700) return [1.0, Math.max(0, (t - 1000) / 2700), 0.0];
  if (t < 5500) return [1.0, 0.39 + (0.61 * (t - 3700)) / 1800, Math.max(0, (t - 3700) / 1800)];
  if (t < 7500) return [1.0 - (0.2 * (t - 5500)) / 2000, 1.0, 0.6 + (0.4 * (t - 5500)) / 2000];
  const hot = Math.min(1, (t - 7500) / 10000);
  return [0.7 - 0.3 * hot, 0.8 + 0.2 * hot, 1.0];
}

/**
 * The warm "Andromeda" palette shared by dust and stars, keyed on where the
 * particle sits. Returns `null` for ordinary disk material, which each
 * population colours its own way.
 */
export function regionColor(r: number, armStrength: number, random: Random): Rgb255 | null {
  const { coreRadius, diskRadius } = GALAXY;

  if (r <= coreRadius * 1.5) {
    // Core: deep yellow/orange.
    return [255, uniformInt(random, 210, 16), uniformInt(random, 80, 61)];
  }
  if (armStrength > 1.0) {
    const roll = random();
    // Arms: mostly warm yellow/orange, some yellow-white, rare blue-white, very rare pink HII.
    if (roll < 0.7) return [255, uniformInt(random, 210, 31), uniformInt(random, 100, 81)];
    if (roll < 0.9) return [255, uniformInt(random, 240, 6), uniformInt(random, 180, 41)];
    if (roll < 0.95) return [uniformInt(random, 180, 41), uniformInt(random, 210, 31), 255];
    return [255, uniformInt(random, 180, 41), uniformInt(random, 220, 36)];
  }
  if (r > diskRadius * 0.9) {
    // Outer halo: faint and reddish.
    return [uniformInt(random, 150, 51), uniformInt(random, 120, 61), uniformInt(random, 120, 61)];
  }
  if (armStrength < 0.3) {
    // Between the arms: brown/tan.
    return [uniformInt(random, 120, 61), uniformInt(random, 90, 51), uniformInt(random, 60, 41)];
  }
  return null;
}

/**
 * Light mode draws with normal alpha blending on a near-white background, so
 * every colour is darkened. Near-white stars get a random dark "cosmic" tint
 * each frame, which reads as a subtle shimmer.
 */
export function toLightMode(base: Rgb255, [r, g, b]: Rgb, random: Random): Rgb {
  const [baseR, baseG, baseB] = base;

  if (baseR > 200 && baseG > 200 && baseB > 200) {
    const roll = random();
    if (roll < 0.5) return [0.0, 0.0, 0.0]; // black
    if (roll < 0.7) return [0.2, 0.0, 0.3]; // dark purple
    if (roll < 0.85) return [0.3, 0.2, 0.0]; // dark amber
    return [0.0, 0.1, 0.3]; // dark blue
  }
  if (baseR > baseG && baseR > baseB) {
    // Red-dominant: dark red with a purple tint.
    return [r * 0.3, g * 0.1, Math.min(0.3, b * 0.2 + 0.1)];
  }
  if (baseB > baseR && baseB > baseG) {
    // Blue-dominant: dark blue with a purple tint.
    return [Math.min(0.2, r * 0.1 + 0.05), g * 0.1, b * 0.4];
  }
  // Yellow/green: dark amber.
  return [r * 0.4, g * 0.3, b * 0.1];
}
