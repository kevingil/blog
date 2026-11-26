import { GALAXY, NEBULA, POPULATION } from './config';
import { regionColor, temperatureToRgb, type Rgb255 } from './palette';
import { uniform, uniformInt, type Random } from './random';
import { DUST_ARM_WIDTHS, STAR_ARM_WIDTHS, spiralStrength } from './spiral';

/**
 * - `nebula`: large, faint, wispy smoke filling the core and arms.
 * - `bulge`: soft glowing gas at the centre; it breathes slowly.
 * - `nucleus`: one very bright point at the exact centre.
 * - `dust`: small diffuse specks of gas and dust tracing the arms.
 * - `star`: stars, concentrated on the arms.
 * - `background`: unrelated field stars spread across the whole sky.
 */
export type ParticleKind = 'nebula' | 'bulge' | 'nucleus' | 'dust' | 'star' | 'background';

export interface Particle {
  kind: ParticleKind;
  /** Position in the galaxy's own plane, before rotation and projection. */
  x: number;
  y: number;
  z: number;
  /** Polar form of (x, y), cached because rotation needs it every frame. */
  radius: number;
  angle: number;
  /** Radians per unit of simulation time. */
  angularSpeed: number;
  /** Base point size in px at zoom 1. */
  size: number;
  /** 0–1 opacity before depth and theme adjustments. */
  brightness: number;
  color: Rgb255;
  /** Marks stars on spiral arms, which the shader renders with a warm glow. */
  spiralArm: boolean;
  /** 0–1 per-sprite offset into the smoke texture, so no two wisps match. */
  seed: number;
}

const TWO_PI = 2 * Math.PI;

function particle(
  kind: ParticleKind,
  x: number,
  y: number,
  z: number,
  size: number,
  brightness: number,
  color: Rgb255,
  spiralArm = false,
  seed = 0,
): Particle {
  const radius = Math.sqrt(x * x + y * y);
  const { coreRadius, coreAngularSpeed } = GALAXY;
  // Solid-body rotation inside the core, flat rotation curve outside it.
  const angularSpeed = radius > coreRadius ? (coreAngularSpeed * coreRadius) / radius : coreAngularSpeed;
  return { kind, x, y, z, radius, angle: Math.atan2(y, x), angularSpeed, size, brightness, color, spiralArm, seed };
}

/**
 * Builds every particle in the scene. The array order is also the draw order,
 * which matters for light mode's non-additive blending: smoke goes first so
 * stars sit on top of it.
 */
export function generateParticles(random: Random = Math.random): Particle[] {
  const particles: Particle[] = [];
  addNebula(particles, random);
  addBulge(particles, random);
  particles.push(particle('nucleus', 0, 0, 0, 8.0, 1.0, [255, 255, 255]));
  addDust(particles, random);
  addStars(particles, random);
  addBackground(particles, random);
  return particles;
}

const CORE_HAZE: Rgb255 = [255, 196, 120];
const INNER_ARM_SMOKE: Rgb255 = [255, 170, 150];
const OUTER_ARM_SMOKE: Rgb255 = [130, 160, 255];
const HII_REGION: Rgb255 = [255, 110, 180];

function mix(a: Rgb255, b: Rgb255, t: number): Rgb255 {
  const channel = (i: 0 | 1 | 2) => Math.round(a[i] + (b[i] - a[i]) * t);
  return [channel(0), channel(1), channel(2)];
}

/**
 * Smoke: a warm haze over the core, and gas along the arms that shifts from
 * rose near the centre to blue at the rim (young hot stars light the outer
 * arms), with rare pink star-forming knots. Each sprite is large and faint;
 * the shader gives it a wispy texture.
 */
function addNebula(out: Particle[], random: Random): void {
  const { coreRadius, diskRadius } = GALAXY;
  const hazeRadius = coreRadius * 2.2;

  for (let i = 0; i < NEBULA.coreHaze; i++) {
    const r = hazeRadius * Math.sqrt(random());
    const theta = random() * TWO_PI;
    const z = (random() - 0.5) * 8;
    const color: Rgb255 = [255, uniformInt(random, CORE_HAZE[1] - 10, 30), uniformInt(random, CORE_HAZE[2] - 20, 50)];
    const size = uniform(random, 60, 50) * (1 - (0.4 * r) / hazeRadius);
    const brightness = uniform(random, 0.1, 0.08);
    out.push(particle('nebula', r * Math.cos(theta), r * Math.sin(theta), z, size, brightness, color, false, random()));
  }

  const outerRadius = diskRadius * 1.05;
  for (let i = 0; i < NEBULA.armCandidates; i++) {
    const r = coreRadius + (outerRadius - coreRadius) * Math.sqrt(random());
    const theta = random() * TWO_PI;
    const arm = spiralStrength(r, theta, DUST_ARM_WIDTHS, 'max', random);
    // Mostly on the arms, with a thin veil between them.
    const armWeight = Math.min(1, arm / 2.5);
    if (random() > 0.04 + armWeight * 0.9) continue;

    const outward = Math.min(1, Math.max(0, (r - coreRadius * 1.5) / (diskRadius * 0.7)));
    const color = arm > 1.5 && random() < 0.08 ? HII_REGION : mix(INNER_ARM_SMOKE, OUTER_ARM_SMOKE, outward);
    const z = (random() - 0.5) * 12 * Math.exp(-r / diskRadius);
    const size = uniform(random, 30, 35) * (0.6 + 0.4 * armWeight);
    const brightness = uniform(random, 0.1, 0.1) * (0.4 + 0.6 * armWeight);
    out.push(particle('nebula', r * Math.cos(theta), r * Math.sin(theta), z, size, brightness, color, false, random()));
  }
}

/** A flattened sphere of glowing gas, white-hot at the centre and yellow at the edge. */
function addBulge(out: Particle[], random: Random): void {
  const cloudRadius = GALAXY.coreRadius * 2.5;

  for (let i = 0; i < POPULATION.bulge; i++) {
    // A power below 1 pushes points outward, softening the edge.
    const r = cloudRadius * Math.pow(random(), 0.6);
    const theta = random() * TWO_PI;
    const phi = Math.acos(2 * random() - 1);
    const x = r * Math.sin(phi) * Math.cos(theta);
    const y = r * Math.sin(phi) * Math.sin(theta);
    const z = r * Math.cos(phi) * 0.3;

    const edge = r / cloudRadius; // 0 at the centre, 1 at the rim
    const falloff = 1.0 - Math.pow(edge, 3);
    const t = 1.0 - falloff;
    const green = Math.floor(242 * t + 255 * (1 - t) - random() * 20 - edge * 20);
    const blue = Math.floor(220 * (1 - t));

    const brightness = uniform(random, 0.2, 0.3) * (0.3 + falloff * 0.7);
    const size = uniform(random, 1.2, 2.5) * (0.8 + falloff * 0.4);
    out.push(particle('bulge', x, y, z, size, brightness, [255, green, blue]));
  }
}

/** Faint, large, diffuse particles that trace the arms as gas and dust lanes. */
function addDust(out: Particle[], random: Random): void {
  const { coreRadius, diskRadius } = GALAXY;

  for (let i = 0; i < POPULATION.dustCandidates; i++) {
    const inDisk = random() < 0.85;
    const v = random();
    const r = inDisk
      ? coreRadius + (diskRadius - coreRadius) * Math.sqrt(v)
      : diskRadius + diskRadius * 0.4 * Math.pow(v, 3);
    const baseTheta = random() * TWO_PI;

    const arm = spiralStrength(r, baseTheta, DUST_ARM_WIDTHS, 'max', random);
    // Keep most candidates either way; arms keep slightly more.
    const keep = arm < 0.2 ? 0.65 + arm * 0.2 : 0.75 + arm * 0.25;
    if (random() > keep) continue;

    const theta = baseTheta + ((random() - 0.5) * 20) / Math.max(r, 15);
    const thickness = 18 * Math.exp(-r / (diskRadius * 0.8));
    const z = (random() - 0.5) * Math.max(thickness, 2);
    const color = regionColor(r, arm, random) ?? [
      uniformInt(random, 60, 90),
      uniformInt(random, 25, 55),
      uniformInt(random, 10, 40),
    ];
    const brightness = uniform(random, 0.05, 0.15);
    const size = uniform(random, 1.2, 3.8);
    out.push(particle('dust', r * Math.cos(theta), r * Math.sin(theta), z, size, brightness, color));
  }
}

/** Stars, kept with a probability that strongly favours the spiral arms. */
function addStars(out: Particle[], random: Random): void {
  const { coreRadius, diskRadius } = GALAXY;
  const maxRadius = diskRadius * 1.8;

  for (let i = 0; i < POPULATION.starCandidates; i++) {
    const inDisk = random() < 0.8;
    const v = random();
    const r = inDisk
      ? coreRadius + (diskRadius - coreRadius) * Math.pow(v, 0.8)
      : diskRadius + (maxRadius - diskRadius) * Math.pow(v, 5);
    const baseTheta = random() * TWO_PI;

    const arm = spiralStrength(r, baseTheta, STAR_ARM_WIDTHS, 'sum', random);
    if (random() > starKeepProbability(r, arm)) continue;

    // Scatter around the arm, a little more where the arm is strong.
    const scatter = Math.min(12, r * 0.025);
    const jitter = (random() - 0.5) * scatter;
    const armJitter = arm > 0.1 ? (random() - 0.5) * scatter * 0.6 * arm : 0;
    const theta = baseTheta + (jitter + armJitter) / Math.max(r, 20);

    const scaleHeight = 10 * Math.exp(-r / (diskRadius * 0.4));
    const z = (random() - 0.5) * Math.max(scaleHeight, 1);

    const { temperature, brightness: intrinsicBrightness } = stellarPopulation(r, arm, random);
    let brightness = intrinsicBrightness;
    if (r <= diskRadius) {
      brightness *= Math.exp(-Math.pow(r / (diskRadius * 0.7), 1.8));
    }

    // Hotter, brighter stars draw larger.
    const temperatureScale = Math.max(0.3, Math.min(2.2, temperature / 6000));
    const size = uniform(random, 0.15, 0.9) * (0.2 + brightness * 1.8) * temperatureScale;

    const color = regionColor(r, arm, random) ?? toRgb255(temperatureToRgb(temperature));
    out.push(
      particle(
        'star',
        r * Math.cos(theta),
        r * Math.sin(theta),
        z,
        Math.max(size, 0.05),
        Math.min(brightness, 1),
        color,
        arm > 0.3,
      ),
    );
  }
}

/** Dense core, arms that dominate the disk, and an almost empty halo. */
function starKeepProbability(r: number, arm: number): number {
  const { coreRadius, diskRadius } = GALAXY;

  if (r <= coreRadius) {
    return 0.6 + arm * 0.3;
  }
  if (r <= diskRadius) {
    const falloff = Math.exp(-Math.pow((r - coreRadius) / (diskRadius - coreRadius), 1.2));
    return 0.005 * falloff + arm * 8.0 * falloff;
  }
  const falloff = Math.exp(-Math.pow((r - diskRadius) / (diskRadius * 0.15), 2));
  return 0.002 * falloff + arm * 2.0 * falloff;
}

/** Temperature (K) and brightness for a star, by region and arm strength. */
function stellarPopulation(r: number, arm: number, random: Random): { temperature: number; brightness: number } {
  const { coreRadius, diskRadius } = GALAXY;
  const star = (tMin: number, tSpan: number, bMin: number, bSpan: number) => {
    const temperature = uniform(random, tMin, tSpan);
    return { temperature, brightness: uniform(random, bMin, bSpan) };
  };

  if (r <= coreRadius * 2) {
    // Mostly old red giants, with some hot young stars.
    return random() < 0.7 ? star(3000, 2000, 0.7, 0.3) : star(15000, 15000, 0.9, 0.1);
  }
  if (r <= diskRadius) {
    if (arm > 2.0) return star(15000, 20000, 0.9, 0.1); // blue supergiants
    if (arm > 1.0) return star(10000, 15000, 0.7, 0.2); // blue giants
    if (arm > 0.3) return star(6000, 8000, 0.5, 0.3); // white stars
    // Between the arms: cool dust, old orange stars, and dim red dwarfs.
    if (random() < 0.3) return star(1500, 1000, 0.08, 0.12);
    if (random() < 0.5) return star(3500, 1500, 0.04, 0.08);
    return star(2800, 1200, 0.02, 0.06);
  }
  // Halo: practically invisible.
  const temperature = uniform(random, 2500, 1000);
  const brightness = Math.exp(-(r - diskRadius) / (diskRadius * 0.1)) * uniform(random, 0.001, 0.01);
  return { temperature, brightness };
}

/** Distant field stars, unrelated to the galaxy, filling the sky. */
function addBackground(out: Particle[], random: Random): void {
  for (let i = 0; i < POPULATION.background; i++) {
    const x = (random() - 0.5) * 3000;
    const y = (random() - 0.5) * 2000;
    const z = (random() - 0.5) * 1000;
    const brightness = uniform(random, 0.08, 0.15);
    const size = uniform(random, 0.8, 1.5);
    const roll = random();
    const color: Rgb255 = roll < 0.6 ? [255, 255, 255] : roll < 0.8 ? [255, 250, 200] : [200, 220, 255];
    out.push(particle('background', x, y, z, size, brightness, color));
  }
}

function toRgb255([r, g, b]: readonly [number, number, number]): Rgb255 {
  return [Math.round(r * 255), Math.round(g * 255), Math.round(b * 255)];
}
