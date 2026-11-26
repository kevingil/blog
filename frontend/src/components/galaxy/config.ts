/**
 * Tunable constants for the spiral galaxy. The values were tuned by eye to
 * resemble Andromeda (M31) seen almost edge-on; distances are in "galaxy
 * units", which map 1:1 to CSS pixels at zoom 1.
 */

export const GALAXY = {
  /** Radius of the bright bulge at the centre. */
  coreRadius: 50,
  /** Radius of the main stellar disk. */
  diskRadius: 350,
  /** Tilt away from face-on (80°). */
  inclination: (80 * Math.PI) / 180,
  /** Position angle of the major axis on screen (10°). */
  orientation: (10 * Math.PI) / 180,
  /** Angular speed inside the core; falls off as 1/r outside it (flat rotation curve). */
  coreAngularSpeed: 0.0001,
} as const;

/** How many candidates each population draws before density thresholding. */
export const POPULATION = {
  bulge: 500,
  dustCandidates: 1400,
  starCandidates: 2000,
  background: 2000,
} as const;

/**
 * Interstellar gas, built from many faint puffs scattered in 3D so it has
 * real thickness and parallax rather than being painted on.
 */
export const NEBULA = {
  /** Puffs in the thick, rounded haze around the bulge. */
  coreHazePuffs: 360,
  /** Cloud complexes along the arms; each is a clump of puffs. */
  cloudComplexes: 240,
  puffsPerComplex: { min: 6, span: 9 },
  /** Absorbing puffs tracing the inner edge of each main arm. */
  dustLanePuffsPerArm: 260,
} as const;

/** Dark mode colour saturation (1 = as generated, 0 = greyscale). */
export const SATURATION = 0.8;

/**
 * Stars are drawn as a point-spread function: a sharp core plus a faint wide
 * halo. The sprite is `min + perBrightness × alpha` times the core size.
 */
export const STAR_HALO = {
  min: 2.5,
  perBrightness: 3.5,
} as const;

/** Cap on device pixels per CSS pixel, to bound fill cost on dense screens. */
export const MAX_PIXEL_RATIO = 2;

export const ANIMATION = {
  /** Simulation time advanced per wall-clock millisecond. */
  timePerMs: 0.16,
  /** Overall playback speed multiplier for galactic rotation. */
  rotationSpeed: 3,
} as const;

export const CAMERA = {
  /** Distance from the camera to the galaxy centre, for perspective. */
  distance: 10000,
  /** Particles this far outside the canvas are still drawn so glows do not pop. */
  cullMargin: 50,
} as const;

export const POINTER_TILT = {
  /** Largest pitch/yaw the pointer can apply (2.5°). */
  maxAngle: (2.5 * Math.PI) / 180,
  /** Fraction of the remaining distance to the target tilt covered each frame. */
  easing: 0.005,
} as const;

export interface GalaxyLayout {
  /** Scale applied to the whole galaxy. */
  zoom: number;
  /** Horizontal offset of the centre from the middle of the canvas, in px. */
  offsetX: number;
  /** Vertical position of the centre from the top of the canvas, in px. */
  offsetY: number;
}

/** Responsive placement, chosen by viewport width only so the hero never jumps vertically. */
const LAYOUTS: ReadonlyArray<{ maxWidth: number; layout: GalaxyLayout }> = [
  { maxWidth: 640, layout: { zoom: 0.6, offsetX: 40, offsetY: 350 } },
  { maxWidth: 1024, layout: { zoom: 0.8, offsetX: 100, offsetY: 345 } },
  { maxWidth: Infinity, layout: { zoom: 1.0, offsetX: 175, offsetY: 340 } },
];

export function layoutForWidth(width: number): GalaxyLayout {
  return LAYOUTS.find((entry) => width <= entry.maxWidth)!.layout;
}

export type ThemeMode = 'light' | 'dark';

export const BACKGROUND: Record<ThemeMode, readonly [number, number, number]> = {
  light: [0.98, 0.98, 1.0],
  dark: [0.02, 0.02, 0.06],
};
