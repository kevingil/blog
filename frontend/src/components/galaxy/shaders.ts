import { NEBULA } from './config';
import { SPRITE } from './frame';

const SPRITE_DEFINES = Object.entries(SPRITE)
  .map(([name, value]) => `#define SPRITE_${name.toUpperCase()} ${value.toFixed(1)}`)
  .join('\n');

/** Positions and sizes arrive in CSS pixels (origin top-left). */
export const VERTEX_SHADER = /* glsl */ `
  attribute vec2 a_position;
  attribute float a_size;
  attribute vec3 a_color;
  attribute float a_alpha;
  attribute float a_kind;
  attribute float a_extra;

  uniform vec2 u_resolution;
  uniform float u_pixelRatio;
  uniform float u_maxPointSize;

  varying vec3 v_color;
  varying float v_alpha;
  varying float v_kind;
  varying float v_extra;

  void main() {
    vec2 clip = (a_position / u_resolution) * 2.0 - 1.0;
    gl_Position = vec4(clip.x, -clip.y, 0.0, 1.0);
    gl_PointSize = min(a_size * u_pixelRatio, u_maxPointSize);
    v_color = a_color;
    v_alpha = a_alpha;
    v_kind = a_kind;
    v_extra = a_extra;
  }
`;

/**
 * Draws each point sprite according to its kind:
 * - stars: a point-spread function, a sharp (white-hot in dark mode) core
 *   inside a faint, wide halo, as a telescope or the eye sees a star;
 * - smoke: a disk-aligned ellipse of domain-warped noise, so overlapping
 *   sprites read as wisps of gas rather than round blobs;
 * - gas and dust: soft Gaussian glows.
 */
export const FRAGMENT_SHADER = /* glsl */ `
  #ifdef GL_FRAGMENT_PRECISION_HIGH
  precision highp float;
  #else
  precision mediump float;
  #endif

  ${SPRITE_DEFINES}

  uniform float u_time;
  uniform float u_light;
  uniform float u_diskAngle;
  uniform float u_diskAspect;

  varying vec3 v_color;
  varying float v_alpha;
  varying float v_kind;
  varying float v_extra;

  const vec3 WARM = vec3(1.0, 0.85, 0.3);

  bool isSprite(float kind) {
    return abs(v_kind - kind) < 0.5;
  }

  float hash(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
  }

  float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    vec2 u = f * f * (3.0 - 2.0 * f);
    return mix(
      mix(hash(i), hash(i + vec2(1.0, 0.0)), u.x),
      mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x),
      u.y
    );
  }

  float fbm(vec2 p) {
    float value = 0.0;
    float amplitude = 0.5;
    for (int octave = 0; octave < 4; octave++) {
      value += amplitude * noise(p);
      p = p * 2.03 + vec2(1.7, 9.2);
      amplitude *= 0.5;
    }
    return value;
  }

  vec4 star(vec2 p) {
    // x is 1 at the edge of the core; the sprite extends to x = halo.
    float halo = v_extra;
    float x = length(p) * 2.0 * halo;
    float core = exp(-x * x * 2.5);
    float glow = pow(1.0 + x * x * 1.5, -2.0);
    float fade = 1.0 - smoothstep(halo * 0.5, halo, x);

    vec3 color = v_color;
    if (isSprite(SPRITE_ARMSTAR)) color = mix(color, WARM, 0.25 * glow);
    if (isSprite(SPRITE_NUCLEUS)) color = mix(color, WARM, 0.4 * glow);
    // Bright cores saturate to white, leaving the colour in the halo.
    if (u_light < 0.5) color = mix(color, vec3(1.0), core * 0.5);
    return vec4(color, v_alpha * (core + glow * 0.18) * fade);
  }

  vec4 smoke(vec2 p) {
    // Squash the sprite into the disk's on-screen ellipse.
    float c = cos(u_diskAngle);
    float s = sin(u_diskAngle);
    vec2 q = vec2(c * p.x + s * p.y, -s * p.x + c * p.y);
    q.y /= u_diskAspect;
    float d = length(q) * 2.0;
    if (d >= 1.0) return vec4(0.0);

    vec2 uv = q * 2.4 + v_extra * 37.0;
    float t = u_time * ${NEBULA.churn.toFixed(6)};
    float warp = fbm(uv + t);
    float density = fbm(uv + vec2(warp, -warp) * 1.8 - t * 0.5);
    float wisps = smoothstep(0.3, 0.75, density);
    float body = 1.0 - smoothstep(0.1, 1.0, d);
    return vec4(v_color, v_alpha * body * (0.25 + 0.75 * wisps));
  }

  // Gas and dust: soft Gaussian glows that reach zero at the sprite edge.
  vec4 gas(float dist) {
    float d2 = dist * dist * 16.0;
    float fade = 1.0 - smoothstep(0.3, 0.5, dist);
    return vec4(v_color, v_alpha * (exp(-d2 * 2.0) + exp(-d2 * 0.5) * 0.5) * fade);
  }

  vec4 dust(float dist) {
    float d2 = dist * dist * 16.0;
    float fade = 1.0 - smoothstep(0.3, 0.5, dist);
    return vec4(v_color, v_alpha * (exp(-d2 * 1.2) * 1.5 + exp(-d2 * 0.35) * 0.9) * fade);
  }

  void main() {
    vec2 p = gl_PointCoord - 0.5;
    float dist = length(p);
    if (dist > 0.5) discard;

    vec4 color;
    if (isSprite(SPRITE_SMOKE)) {
      color = smoke(p);
    } else if (isSprite(SPRITE_GAS)) {
      color = gas(dist);
    } else if (isSprite(SPRITE_DUST)) {
      color = dust(dist);
    } else {
      color = star(p);
    }
    if (color.a < 0.002) discard;
    gl_FragColor = color;
  }
`;
