import { describe, expect, test } from 'bun:test';
import { GALAXY, layoutForWidth, NEBULA, POPULATION, STAR_HALO } from './config';
import { diskEllipse, FLOATS_PER_VERTEX, projectParticle, SPRITE, writeFrame, type FrameState } from './frame';
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
    expect(counts.nebula).toBeGreaterThan(NEBULA.coreHaze);
    expect(counts.nebula).toBeLessThan(NEBULA.coreHaze + NEBULA.armCandidates);
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

  test('draws smoke first and the nucleus after the bulge, so stars sit on top', () => {
    const particles = generateParticles(seededRandom(5));
    const firstNonSmoke = particles.findIndex((p) => p.kind !== 'nebula');
    expect(particles.slice(0, firstNonSmoke).every((p) => p.kind === 'nebula')).toBe(true);
    expect(particles[firstNonSmoke].kind).toBe('bulge');
    expect(particles[firstNonSmoke + POPULATION.bulge].kind).toBe('nucleus');
    expect(particles.slice(firstNonSmoke).some((p) => p.kind === 'nebula')).toBe(false);
  });

  test('gives every smoke sprite its own texture seed', () => {
    const seeds = generateParticles(seededRandom(9))
      .filter((p) => p.kind === 'nebula')
      .map((p) => p.seed);
    expect(new Set(seeds).size).toBe(seeds.length);
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

describe('diskEllipse', () => {
  test('matches the outline of a projected ring in the disk', () => {
    for (const tilt of [{}, { pitch: 0.04, yaw: -0.03 }]) {
      const state = frameState(tilt);
      const centre = { x: state.width / 2 + state.layout.offsetX, y: state.layout.offsetY };
      let farthest = { distance: 0, angle: 0 };
      for (let i = 0; i < 720; i++) {
        const angle = (i / 720) * 2 * Math.PI;
        const ring = { kind: 'dust', radius: 100, angle, angularSpeed: 0, x: 0, y: 0, z: 0 } as Particle;
        const point = projectParticle(ring, state);
        const distance = Math.hypot(point.x - centre.x, point.y - centre.y);
        if (distance > farthest.distance) farthest = { distance, angle: Math.atan2(point.y - centre.y, point.x - centre.x) };
      }
      const { angle } = diskEllipse(state);
      // Same axis, modulo π.
      expect(Math.abs(Math.sin(angle - farthest.angle))).toBeLessThan(0.02);
    }
  });

  test('never flattens smoke below the minimum thickness', () => {
    const { aspect } = diskEllipse(frameState());
    expect(aspect).toBeGreaterThanOrEqual(NEBULA.minAspect);
    expect(aspect).toBeLessThanOrEqual(1);
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

  test('sizes star sprites to hold a halo that grows with brightness', () => {
    const out = buffer();
    const count = writeFrame(particles, frameState(), out);
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
