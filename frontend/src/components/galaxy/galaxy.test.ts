import { describe, expect, test } from 'bun:test';
import { GALAXY, layoutForWidth, POPULATION } from './config';
import { FLOATS_PER_VERTEX, projectParticle, writeFrame, type FrameState } from './frame';
import { temperatureToRgb } from './palette';
import { generateParticles, type Particle } from './particles';
import { seededRandom } from './random';
import { wrapAngle } from './spiral';

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
    expect(counts.bulge).toBe(POPULATION.bulge);
    expect(counts.nucleus).toBe(1);
    expect(counts.background).toBe(POPULATION.background);
    expect(counts.dust).toBeGreaterThan(0);
    expect(counts.dust).toBeLessThanOrEqual(POPULATION.dustCandidates);
    expect(counts.star).toBeGreaterThan(0);
    expect(counts.star).toBeLessThan(POPULATION.starCandidates);
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

  test('draws the bulge first, so the nucleus glows on top of it', () => {
    const particles = generateParticles(seededRandom(5));
    expect(particles[POPULATION.bulge].kind).toBe('nucleus');
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

  test('culls particles that fall off screen', () => {
    const count = writeFrame(particles, frameState(), buffer());
    expect(count).toBeGreaterThan(0);
    expect(count).toBeLessThan(particles.length);
    expect(writeFrame(particles, frameState({ width: 10, height: 10 }), buffer())).toBeLessThan(count);
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
    const count = writeFrame(particles, frameState({ theme: 'light' }), out, seededRandom(1));
    for (let i = 0; i < count; i++) {
      const base = i * FLOATS_PER_VERTEX;
      for (const channel of out.subarray(base + 3, base + 6)) expect(channel).toBeLessThanOrEqual(0.41);
      expect(out[base + 6]).toBeLessThanOrEqual(1);
      expect(out[base + 2]).toBeGreaterThanOrEqual(1);
    }
  });
});
