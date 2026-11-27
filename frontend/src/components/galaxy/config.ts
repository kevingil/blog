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
  /** Multiplies the vertical spread of every disk population; higher looks thicker. */
  thickness: 2.2,
  /** Height of the bulge relative to its width (1 would be a sphere). */
  bulgeFlattening: 0.55,
} as const;

/** How many particles each population draws (dust is thinned afterwards). */
export const POPULATION = {
  bulge: 500,
  dustCandidates: 1400,
  /** The smooth disk of old stars; the arms barely show in it. */
  oldStars: 1700,
  background: 2000,
} as const;

/** Young, hot stars born in clusters where gas crosses an arm. */
export const YOUNG_STARS = {
  clusters: 540,
  perCluster: { min: 2, span: 6 },
} as const;

/**
 * The spiral arms are a density wave: a pattern that turns rigidly and more
 * slowly than the material, which flows through it. Gas brightens as it
 * enters an arm, dust collects on the upstream edge, and young stars light
 * up just past it and fade as they drift downstream. Angles are radians in
 * the pattern's frame.
 */
export const DENSITY_WAVE = {
  /** Radius where material and pattern move together, as a multiple of the disk radius. */
  corotation: 1.25,
  /** Gas brightness between the arms, as a fraction of its brightness on them. */
  interArmGlow: 0.08,
  /** Gas lanes are this fraction of an arm's full width, so they read as long thin spirals. */
  gasLaneWidth: 0.6,
  /** Dust lanes sit this far upstream of the arm centre, and are this wide. */
  laneOffset: -0.22,
  laneWidth: 0.15,
  /** Stars are born this far from the arm centre, and fade over this many radians downstream. */
  birthOffset: -0.06,
  youthDecay: 0.7,
} as const;

/**
 * Interstellar gas, built from many faint puffs scattered in 3D so it has
 * real thickness and parallax rather than being painted on.
 */
export const NEBULA = {
  /** Puffs in the thick, rounded haze around the bulge. */
  coreHazePuffs: 280,
  /** Gas streamers spread over the disk; each is a thin arc of puffs along its orbit. */
  streamers: 300,
  streamerArc: { min: 0.5, span: 0.6 },
  puffsPerStreamer: { min: 9, span: 6 },
  /** Absorbing puffs over the disk; only those in a lane show at any moment. */
  dustLanePuffs: 4200,
} as const;

/** Dark mode colour saturation (1 = as generated, 0 = greyscale). */
export const SATURATION = 0.8;

/**
 * Stars are drawn as a point-spread function: a sharp core plus a faint wide
 * halo. The sprite is `min + perBrightness × alpha` times the core size.
 */
export const STAR_HALO = {
  min: 3.5,
  perBrightness: 5,
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
