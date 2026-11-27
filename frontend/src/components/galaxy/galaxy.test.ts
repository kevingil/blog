import { describe, expect, test } from 'bun:test';
import { DENSITY_WAVE, GALAXY, layoutForWidth, NEBULA, POPULATION, STAR_HALO } from './config';
import { FLOATS_PER_VERTEX, projectParticle, SPRITE, writeFrame, type FrameState } from './frame';
import { temperatureToRgb } from './palette';
import { generateParticles, type Particle } from './particles';
import { seededRandom } from './random';
import { armAngles, wrapAngle } from './spiral';
import { armGlow, laneDepth, PATTERN_SPEED, patternAngle, youth } from './wave';

function frameState(overrides: Partial<FrameState> = {}): FrameState {
  return {
    time: 0,
    pitch: 0,
    yaw: 0,
    layout: layoutForWidth(1280),
    width: 1280,
    height: 800,
    theme: 'dark',
    ...overrides,
  };
}

function countKinds(particles: Particle[]) {
  const counts: Record<string, number> = {};
  for (const p of particles) counts[p.kind] = (counts[p.kind] ?? 0) + 1;
  return counts;
}

describe('generateParticles', () => {
  test('is reproducible for a given seed', () => {
    expect(generateParticles(seededRandom(7))).toEqual(generateParticles(seededRandom(7)));
    expect(generateParticles(seededRandom(7))).not.toEqual(generateParticles(seededRandom(8)));
  });

  test('builds every population, with arm stars and dust thresholded', () => {
    const counts = countKinds(generateParticles(seededRandom(1)));
    expect(counts.haze).toBe(NEBULA.coreHazePuffs);
    expect(counts.cloud).toBeGreaterThan(NEBULA.cloudComplexes * NEBULA.puffsPerComplex.min);
    expect(counts.dustLane).toBe(NEBULA.dustLanePuffs);
    expect(counts.youngStar).toBeGreaterThan(0);
    expect(counts.bulge).toBe(POPULATION.bulge);
    expect(counts.nucleus).toBe(1);
    expect(counts.background).toBe(POPULATION.background);
    expect(counts.dust).toBeGreaterThan(0);
    expect(counts.dust).toBeLessThanOrEqual(POPULATION.dustCandidates);
    expect(counts.star).toBe(POPULATION.oldStars);
  });

  test('keeps sizes, brightness and colours in range', () => {
    for (const p of generateParticles(seededRandom(3))) {
      expect(p.size).toBeGreaterThan(0);
      expect(p.brightness).toBeGreaterThanOrEqual(0);
      expect(p.brightness).toBeLessThanOrEqual(1);
      for (const channel of p.color) {
        expect(Number.isInteger(channel)).toBe(true);
        expect(channel).toBeGreaterThanOrEqual(0);
        expect(channel).toBeLessThanOrEqual(255);
      }
    }
  });

  test('draws gas first and dust lanes last, so stars sit on the gas and dust can darken it', () => {
    const kinds = generateParticles(seededRandom(5)).map((p) => p.kind);
    const isGas = (kind: string) => kind === 'haze' || kind === 'cloud';
    const firstNonGas = kinds.findIndex((kind) => !isGas(kind));
    const firstLane = kinds.indexOf('dustLane');
    expect(kinds.slice(0, firstNonGas).every(isGas)).toBe(true);
    expect(kinds.slice(firstLane).every((kind) => kind === 'dustLane')).toBe(true);
    expect(kinds.slice(firstNonGas, firstLane).some(isGas)).toBe(false);
    expect(kinds[firstNonGas]).toBe('bulge');
    expect(kinds[firstNonGas + POPULATION.bulge]).toBe('nucleus');
  });

  test('gives only gas, dust lanes and young stars arm angles for the density wave', () => {
    for (const p of generateParticles(seededRandom(6))) {
      expect(p.arms !== null).toBe(p.kind === 'cloud' || p.kind === 'dustLane' || p.kind === 'youngStar');
    }
  });

  test('gives the gas real thickness around the core', () => {
    const haze = generateParticles(seededRandom(4)).filter((p) => p.kind === 'haze' && p.radius < GALAXY.coreRadius);
    const spread = Math.sqrt(haze.reduce((sum, p) => sum + p.z * p.z, 0) / haze.length);
    expect(spread).toBeGreaterThan(GALAXY.coreRadius * 0.4);
  });
});

describe('wrapAngle', () => {
  test('maps any angle into [-π, π) without changing its direction', () => {
    for (const angle of [-20, -Math.PI, -1, 0, 1, Math.PI, 7, 50]) {
      const wrapped = wrapAngle(angle);
      expect(wrapped).toBeGreaterThanOrEqual(-Math.PI);
      expect(wrapped).toBeLessThan(Math.PI);
      expect(Math.cos(wrapped)).toBeCloseTo(Math.cos(angle), 9);
      expect(Math.sin(wrapped)).toBeCloseTo(Math.sin(angle), 9);
    }
  });
});

describe('temperatureToRgb', () => {
  test('runs from red for cool stars to blue-white for hot ones', () => {
    const [coolR, , coolB] = temperatureToRgb(3000);
    const [hotR, , hotB] = temperatureToRgb(30000);
    expect(coolR).toBe(1);
    expect(coolB).toBe(0);
    expect(hotB).toBe(1);
    expect(hotR).toBeLessThan(coolR);
  });
});

describe('layoutForWidth', () => {
  test('shrinks and recentres the galaxy on narrow screens', () => {
    expect(layoutForWidth(375).zoom).toBe(0.6);
    expect(layoutForWidth(640).zoom).toBe(0.6);
    expect(layoutForWidth(641).zoom).toBe(0.8);
    expect(layoutForWidth(1024).zoom).toBe(0.8);
    expect(layoutForWidth(1920).zoom).toBe(1);
  });
});

describe('density wave', () => {
  const r = GALAXY.diskRadius * 0.5;
  const arms = armAngles(r);
  const mainArm = arms[0];

  test('turns more slowly than the material everywhere in the disk', () => {
    const slowestDiskSpeed = (GALAXY.coreAngularSpeed * GALAXY.coreRadius) / GALAXY.diskRadius;
    expect(PATTERN_SPEED).toBeGreaterThan(0);
    expect(PATTERN_SPEED).toBeLessThan(slowestDiskSpeed);
  });

  test('moves material forward through the pattern over time', () => {
    const orbiter = { angle: 1, angularSpeed: GALAXY.coreAngularSpeed * 0.2 };
    expect(patternAngle(orbiter, 0)).toBe(1);
    expect(patternAngle(orbiter, 1e5)).toBeGreaterThan(1);
  });

  test('lights gas on a main arm and leaves it dim between arms', () => {
    expect(armGlow(arms, mainArm)).toBeCloseTo(1, 5);
    const between = Math.min(...Array.from({ length: 64 }, (_, i) => armGlow(arms, (i / 64) * 2 * Math.PI)));
    expect(between).toBeGreaterThanOrEqual(DENSITY_WAVE.interArmGlow);
    expect(between).toBeLessThan(DENSITY_WAVE.interArmGlow + 0.05);
  });

  test('puts dust lanes on the upstream edge of an arm', () => {
    expect(laneDepth(arms, mainArm + DENSITY_WAVE.laneOffset)).toBeCloseTo(1, 5);
    expect(laneDepth(arms, mainArm + 0.5)).toBeLessThan(0.01);
  });

  test('ignites young stars just past an arm and fades them downstream', () => {
    const birth = mainArm + DENSITY_WAVE.birthOffset;
    expect(youth(arms, birth - 0.05)).toBeLessThan(0.05); // not yet born
    expect(youth(arms, birth + 0.2)).toBeGreaterThan(0.6);
    expect(youth(arms, birth + 0.2)).toBeGreaterThan(youth(arms, birth + 1));
    expect(youth(arms, birth + 2.5)).toBeLessThan(0.05);
  });
});

describe('projectParticle', () => {
  test('puts the galaxy centre at the layout offset', () => {
    const nucleus = generateParticles(seededRandom(1)).find((p) => p.kind === 'nucleus')!;
    const state = frameState({ time: 1e6, pitch: 0.03, yaw: -0.02 });
    const point = projectParticle(nucleus, state);
    expect(point.x).toBeCloseTo(state.width / 2 + state.layout.offsetX, 9);
    expect(point.y).toBeCloseTo(state.layout.offsetY, 9);
    expect(point.scale).toBeCloseTo(state.layout.zoom, 9);
  });

  test('rotates the inner disk faster than the outskirts', () => {
    const particles = generateParticles(seededRandom(1));
    const inner = particles.find((p) => p.kind === 'bulge' && p.radius < GALAXY.coreRadius)!;
    const outer = particles.find((p) => p.kind === 'star' && p.radius > GALAXY.diskRadius * 0.8)!;
    expect(inner.angularSpeed).toBeGreaterThan(outer.angularSpeed);
  });
});

describe('writeFrame', () => {
  const particles = generateParticles(seededRandom(11));
  const buffer = () => new Float32Array(particles.length * FLOATS_PER_VERTEX);

  test('puts every absorbing dust lane after the emissive vertices', () => {
    const out = buffer();
    const { total, emissive } = writeFrame(particles, frameState(), out);
    expect(emissive).toBeGreaterThan(0);
    expect(total).toBeGreaterThan(emissive);
    for (let i = 0; i < total; i++) {
      const isLane = out[i * FLOATS_PER_VERTEX + 7] === SPRITE.dustLane;
      expect(isLane).toBe(i >= emissive);
    }
  });

  test('keeps near-side dust lanes darker than far-side ones', () => {
    const state = frameState();
    const inLane = particles.filter((p) => p.kind === 'dustLane' && laneDepth(p.arms!, patternAngle(p, 0)) > 0.9);
    const opacity = (p: Particle) => {
      const out = new Float32Array(FLOATS_PER_VERTEX);
      writeFrame([p], state, out);
      return out[6] / (p.brightness * laneDepth(p.arms!, patternAngle(p, 0)));
    };
    const near = inLane.find((p) => projectParticle(p, state).depth < -150)!;
    const far = inLane.find((p) => projectParticle(p, state).depth > 150)!;
    expect(opacity(near)).toBeGreaterThan(opacity(far));
  });

  test('dark mode does not consume randomness', () => {
    const a = buffer();
    const b = buffer();
    writeFrame(particles, frameState({ time: 500 }), a, seededRandom(1));
    writeFrame(particles, frameState({ time: 500 }), b, seededRandom(2));
    expect(a).toEqual(b);
  });

  test('light mode darkens every colour and keeps alpha in range', () => {
    const out = buffer();
    const { total: count } = writeFrame(particles, frameState({ theme: 'light' }), out, seededRandom(1));
    for (let i = 0; i < count; i++) {
      const base = i * FLOATS_PER_VERTEX;
      for (const channel of out.subarray(base + 3, base + 6)) expect(channel).toBeLessThanOrEqual(0.41);
      expect(out[base + 6]).toBeLessThanOrEqual(1);
      expect(out[base + 2]).toBeGreaterThanOrEqual(1);
    }
  });

  test('sizes star sprites to hold a halo that grows with brightness', () => {
    const out = buffer();
    const { total: count } = writeFrame(particles, frameState(), out);
    let stars = 0;
    for (let i = 0; i < count; i++) {
      const base = i * FLOATS_PER_VERTEX;
      if (out[base + 7] !== SPRITE.star && out[base + 7] !== SPRITE.armStar) continue;
      stars++;
      const halo = out[base + 8];
      expect(halo).toBeCloseTo(STAR_HALO.min + STAR_HALO.perBrightness * Math.min(1, out[base + 6]), 4);
      expect(out[base + 2]).toBeGreaterThanOrEqual(halo - 1e-4);
    }
    expect(stars).toBeGreaterThan(0);
  });
});
