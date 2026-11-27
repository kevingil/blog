import { GALAXY, NEBULA, POPULATION, YOUNG_STARS } from './config';
import { regionColor, temperatureToRgb, type Rgb255 } from './palette';
import { gaussian, truncatedGaussian, uniform, uniformInt, type Random } from './random';
import { armAngles, spiralStrength } from './spiral';

/**
 * - `haze`: thick, rounded glowing gas around the bulge.
 * - `cloud`: gas in the disk; it lights up as it flows through an arm.
 * - `bulge`: soft glowing gas at the centre; it breathes slowly.
 * - `nucleus`: one very bright point at the exact centre.
 * - `dust`: small diffuse specks of gas and dust.
 * - `star`: old stars in a smooth disk.
 * - `youngStar`: hot stars that are born in an arm and fade downstream.
 * - `background`: unrelated field stars spread across the whole sky.
 * - `dustLane`: dark dust; it shows only in the lane on an arm's upstream edge.
 */
export type ParticleKind =
  | 'haze'
  | 'cloud'
  | 'bulge'
  | 'nucleus'
  | 'dust'
  | 'star'
  | 'youngStar'
  | 'background'
  | 'dustLane';

/** Kinds whose brightness follows the density wave. */
const WAVE_KINDS: ReadonlySet<ParticleKind> = new Set(['cloud', 'youngStar', 'dustLane']);

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
  /** 0–1 peak opacity before depth, density-wave and theme adjustments. */
  brightness: number;
  color: Rgb255;
  /** Marks stars that render with a warm glow. */
  spiralArm: boolean;
  /** For particles that respond to the density wave: arm angles at this radius (see `armAngles`). */
  arms: readonly number[] | null;
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
): Particle {
  const radius = Math.sqrt(x * x + y * y);
  const { coreRadius, coreAngularSpeed } = GALAXY;
  // Solid-body rotation inside the core, flat rotation curve outside it.
  const angularSpeed = radius > coreRadius ? (coreAngularSpeed * coreRadius) / radius : coreAngularSpeed;
  return {
    kind,
    x,
    y,
    z,
    radius,
    angle: Math.atan2(y, x),
    angularSpeed,
    size,
    brightness,
    color,
    spiralArm,
    arms: WAVE_KINDS.has(kind) ? armAngles(radius) : null,
  };
}

/** A particle at polar position (r, θ) in the disk. */
function polar(kind: ParticleKind, r: number, theta: number, z: number, size: number, brightness: number, color: Rgb255) {
  return particle(kind, r * Math.cos(theta), r * Math.sin(theta), z, size, brightness, color);
}

/**
 * Builds every particle in the scene. The array order is also the draw order,
 * which matters for blending: glowing gas goes first so stars sit on top of
 * it, and absorbing dust lanes go last so they can darken everything behind.
 */
export function generateParticles(random: Random = Math.random): Particle[] {
  const particles: Particle[] = [];
  addCoreHaze(particles, random);
  addClouds(particles, random);
  addBulge(particles, random);
  particles.push(particle('nucleus', 0, 0, 0, 8.0, 1.0, [255, 255, 255]));
  addDust(particles, random);
  addOldStars(particles, random);
  addYoungStars(particles, random);
  addBackground(particles, random);
  addDustLanes(particles, random);
  return particles;
}

// Muted, dusty tones: real nebulae look grey-beige next to their stars.
const HAZE_WARM: Rgb255 = [230, 206, 170];
const CLOUD_INNER: Rgb255 = [220, 202, 178];
const CLOUD_OUTER: Rgb255 = [176, 186, 204];
const DUST_LANE: Rgb255 = [62, 50, 42];

function mix(a: Rgb255, b: Rgb255, t: number): Rgb255 {
  const channel = (i: 0 | 1 | 2) => Math.round(a[i] + (b[i] - a[i]) * t);
  return [channel(0), channel(1), channel(2)];
}

function jitter(color: Rgb255, random: Random, amount: number): Rgb255 {
  const shift = (random() - 0.5) * 2 * amount;
  const channel = (c: number) => Math.max(0, Math.min(255, Math.round(c + shift)));
  return [channel(color[0]), channel(color[1]), channel(color[2])];
}

/** Pulls a colour `amount` of the way toward its own grey. */
function muted(color: Rgb255, amount: number): Rgb255 {
  const grey = 0.2126 * color[0] + 0.7152 * color[1] + 0.0722 * color[2];
  const channel = (c: number) => Math.round(c + (grey - c) * amount);
  return [channel(color[0]), channel(color[1]), channel(color[2])];
}

function toRgb255([r, g, b]: readonly [number, number, number]): Rgb255 {
  return [Math.round(r * 255), Math.round(g * 255), Math.round(b * 255)];
}

/** 0 at the core edge, 1 toward the rim: drives the warm-to-cool colour shift. */
function outwardness(r: number): number {
  const { coreRadius, diskRadius } = GALAXY;
  return Math.min(1, Math.max(0, (r - coreRadius * 1.5) / (diskRadius * 0.7)));
}

/**
 * Gas is drawn as many large, very faint, heavily overlapping puffs: no
 * single puff is visible, only their summed density, which reads as one
 * continuous cloud. Puffs are kept within a couple of standard deviations so
 * none sits alone at the fringe as a visible blob.
 */
const GAS_SPREAD_LIMIT = 2.2;

/** The bulge is nearly spherical, so its haze is too. */
function addCoreHaze(out: Particle[], random: Random): void {
  const { coreRadius } = GALAXY;
  for (let i = 0; i < NEBULA.coreHazePuffs; i++) {
    const x = truncatedGaussian(random, GAS_SPREAD_LIMIT) * coreRadius * 1.0;
    const y = truncatedGaussian(random, GAS_SPREAD_LIMIT) * coreRadius * 1.0;
    const z = truncatedGaussian(random, GAS_SPREAD_LIMIT) * coreRadius * 0.6;
    const size = uniform(random, 60, 40);
    const brightness = uniform(random, 0.006, 0.004);
    out.push(particle('haze', x, y, z, size, brightness, jitter(HAZE_WARM, random, 8)));
  }
}

/**
 * Gas in long, thin streamers along circular orbits: differential rotation
 * shears any cloud into an arc. Spread evenly over the disk, the streamers
 * light up only where they cross an arm, so together they draw the arms as
 * long continuous spiral lanes, with fine striations along the orbits.
 */
function addClouds(out: Particle[], random: Random): void {
  const { coreRadius, diskRadius } = GALAXY;

  for (let i = 0; i < NEBULA.streamers; i++) {
    const centreR = coreRadius * 1.2 + (diskRadius - coreRadius * 1.2) * Math.sqrt(random());
    const centreTheta = random() * TWO_PI;
    const scaleHeight = (5 + 14 * Math.exp(-centreR / (diskRadius * 0.5))) * GALAXY.thickness;
    const centreZ = truncatedGaussian(random, GAS_SPREAD_LIMIT) * scaleHeight * 0.4;
    const tone = mix(CLOUD_INNER, CLOUD_OUTER, outwardness(centreR));
    // Arc length in radians; long enough to span an arm at any radius.
    const arc = uniform(random, NEBULA.streamerArc.min, NEBULA.streamerArc.span);
    const thickness = uniform(random, 4, 5);

    const puffs = uniformInt(random, NEBULA.puffsPerStreamer.min, NEBULA.puffsPerStreamer.span);
    for (let j = 0; j < puffs; j++) {
      const r = centreR + truncatedGaussian(random, GAS_SPREAD_LIMIT) * thickness;
      const theta = centreTheta + (random() - 0.5) * arc;
      const z = centreZ + truncatedGaussian(random, GAS_SPREAD_LIMIT) * scaleHeight * 0.4;
      const size = uniform(random, 30, 18);
      const brightness = uniform(random, 0.009, 0.006);
      out.push(polar('cloud', r, theta, z, size, brightness, jitter(tone, random, 8)));
    }
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
    const z = r * Math.cos(phi) * GALAXY.bulgeFlattening;

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

/** Small diffuse specks of gas and dust, slightly favouring the arms. */
function addDust(out: Particle[], random: Random): void {
  const { coreRadius, diskRadius } = GALAXY;

  for (let i = 0; i < POPULATION.dustCandidates; i++) {
    const inDisk = random() < 0.85;
    const v = random();
    const r = inDisk
      ? coreRadius + (diskRadius - coreRadius) * Math.sqrt(v)
      : diskRadius + diskRadius * 0.4 * Math.pow(v, 3);
    const baseTheta = random() * TWO_PI;

    const arm = spiralStrength(r, baseTheta, random);
    const keep = arm < 0.2 ? 0.65 + arm * 0.2 : 0.75 + arm * 0.25;
    if (random() > keep) continue;

    const theta = baseTheta + ((random() - 0.5) * 20) / Math.max(r, 15);
    const thickness = 18 * Math.exp(-r / (diskRadius * 0.8)) * GALAXY.thickness;
    const z = (random() - 0.5) * Math.max(thickness, 2);
    const color = regionColor(r, arm, random) ?? [
      uniformInt(random, 60, 90),
      uniformInt(random, 25, 55),
      uniformInt(random, 10, 40),
    ];
    const brightness = uniform(random, 0.04, 0.1);
    const size = uniform(random, 1.2, 3.8);
    out.push(polar('dust', r, theta, z, size, brightness, muted(color, 0.55)));
  }
}

/**
 * Old stars: a smooth exponential disk, thick and bright toward the centre.
 * They have orbited many times, so the arms are smeared out of them.
 */
function addOldStars(out: Particle[], random: Random): void {
  const { coreRadius, diskRadius } = GALAXY;

  for (let i = 0; i < POPULATION.oldStars; i++) {
    const r = coreRadius * 0.6 + Math.min(diskRadius * 1.6, -Math.log(1 - random()) * diskRadius * 0.32);
    const theta = random() * TWO_PI;
    const z = gaussian(random) * (3 + 10 * Math.exp(-r / (diskRadius * 0.35))) * GALAXY.thickness;

    let temperature: number;
    let brightness: number;
    if (r <= coreRadius * 2) {
      // Bulge giants.
      temperature = uniform(random, 3200, 2600);
      brightness = uniform(random, 0.55, 0.4);
    } else if (random() < 0.85) {
      temperature = uniform(random, 3200, 2800);
      brightness = uniform(random, 0.15, 0.3);
    } else {
      temperature = uniform(random, 6000, 3000);
      brightness = uniform(random, 0.3, 0.3);
    }
    brightness *=
      r <= diskRadius
        ? Math.exp(-Math.pow(r / (diskRadius * 0.7), 1.8))
        : 0.15 * Math.exp(-(r - diskRadius) / (diskRadius * 0.15));

    const temperatureScale = Math.max(0.3, Math.min(2.2, temperature / 6000));
    const size = uniform(random, 0.15, 0.9) * (0.2 + brightness * 1.8) * temperatureScale;
    // A mid-range arm strength picks the region palette without arm colours.
    const color = regionColor(r, 0.5, random) ?? toRgb255(temperatureToRgb(temperature));
    out.push(polar('star', r, theta, z, Math.max(size, 0.05), Math.min(brightness, 1), color));
  }
}

/**
 * Hot blue-white stars in small clusters (OB associations), spread over the
 * disk. The density wave decides when each one is visible.
 */
function addYoungStars(out: Particle[], random: Random): void {
  const { coreRadius, diskRadius } = GALAXY;

  for (let c = 0; c < YOUNG_STARS.clusters; c++) {
    const centreR = coreRadius * 1.3 + (diskRadius * 1.02 - coreRadius * 1.3) * Math.sqrt(random());
    const centreTheta = random() * TWO_PI;
    const stars = uniformInt(random, YOUNG_STARS.perCluster.min, YOUNG_STARS.perCluster.span);

    for (let s = 0; s < stars; s++) {
      const r = centreR + gaussian(random) * 3;
      const theta = centreTheta + (gaussian(random) * 8) / centreR;
      const z = gaussian(random) * 2.5 * GALAXY.thickness;
      const temperature = uniform(random, 9000, 16000);
      const brightness = uniform(random, 0.7, 0.3) * Math.exp(-Math.pow(r / (diskRadius * 1.1), 2));
      const temperatureScale = Math.min(2.2, temperature / 6000);
      const size = uniform(random, 0.35, 0.6) * (0.2 + brightness * 1.8) * temperatureScale;
      out.push(polar('youngStar', r, theta, z, size, brightness, toRgb255(temperatureToRgb(temperature))));
    }
  }
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

/**
 * Dark dust spread thinly over the disk. The density wave makes it visible
 * only where it piles up in a lane on an arm's upstream edge, so the lanes
 * persist while the dust itself streams through them.
 */
function addDustLanes(out: Particle[], random: Random): void {
  const { coreRadius, diskRadius } = GALAXY;

  for (let i = 0; i < NEBULA.dustLanePuffs; i++) {
    // Denser toward the centre, like the arms themselves.
    const r = coreRadius * 1.4 + (diskRadius * 0.95 - coreRadius * 1.4) * Math.pow(random(), 1.3);
    const theta = random() * TWO_PI;
    const z = truncatedGaussian(random, GAS_SPREAD_LIMIT) * 3 * GALAXY.thickness;
    const size = uniform(random, 20, 12) * (0.7 + (0.6 * r) / diskRadius);
    const opacity = uniform(random, 0.1, 0.06);
    out.push(polar('dustLane', r, theta, z, size, opacity, DUST_LANE));
  }
}
