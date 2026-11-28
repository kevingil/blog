/** A source of uniform random numbers in [0, 1), such as `Math.random`. */
export type Random = () => number;

/** A float in [min, min + span). */
export function uniform(random: Random, min: number, span: number): number {
  return min + random() * span;
}

/** An integer in [min, min + count). */
export function uniformInt(random: Random, min: number, count: number): number {
  return min + Math.floor(random() * count);
}

/** A standard normal sample (mean 0, sd 1), by the Box–Muller transform. */
export function gaussian(random: Random): number {
  return Math.sqrt(-2 * Math.log(1 - random())) * Math.cos(2 * Math.PI * random());
}

/**
 * A normal sample limited to ±`limit` standard deviations (by resampling), for
 * clouds that should thin out smoothly without stray outliers.
 */
export function truncatedGaussian(random: Random, limit: number): number {
  for (;;) {
    const sample = gaussian(random);
    if (Math.abs(sample) <= limit) return sample;
  }
}

/**
 * A deterministic generator (mulberry32), for tests and reproducible
 * screenshots. Not suitable for anything security related.
 */
export function seededRandom(seed: number): Random {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
