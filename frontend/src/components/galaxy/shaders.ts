import { SATURATION } from './config';
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
  attribute float a_halo;

  uniform vec2 u_resolution;
  uniform float u_pixelRatio;
  uniform float u_maxPointSize;

  varying vec3 v_color;
  varying float v_alpha;
  varying float v_kind;
  varying float v_halo;

  void main() {
    vec2 clip = (a_position / u_resolution) * 2.0 - 1.0;
    gl_Position = vec4(clip.x, -clip.y, 0.0, 1.0);
    gl_PointSize = min(a_size * u_pixelRatio, u_maxPointSize);
    v_color = a_color;
    v_alpha = a_alpha;
    v_kind = a_kind;
    v_halo = a_halo;
  }
`;

/**
 * Draws each point sprite according to its kind. Every profile fades to zero
 * before the sprite's edge, so nothing shows as a hard disc.
 * - Stars: a point-spread function, a soft core (white-hot in dark mode)
 *   inside a faint halo.
 * - Gas, dust, clouds and dust lanes: soft glows of different widths. Volume
 *   comes from where the sprites sit in 3D, not from any texture.
 */
export const FRAGMENT_SHADER = /* glsl */ `
  precision mediump float;

  ${SPRITE_DEFINES}
  #define SATURATION ${SATURATION.toFixed(3)}

  uniform float u_light;
  // 1 when drawing into a layer that is composited with premultiplied alpha.
  uniform float u_premultiply;

  varying vec3 v_color;
  varying float v_alpha;
  varying float v_kind;
  varying float v_halo;

  const vec3 WARM = vec3(1.0, 0.85, 0.3);
  // Dust blocks blue light more than red, so lanes look reddish-brown.
  const vec3 DUST_ABSORPTION = vec3(0.55, 0.75, 0.95);
  const vec3 LUMA = vec3(0.2126, 0.7152, 0.0722);

  bool isSprite(float kind) {
    return abs(v_kind - kind) < 0.5;
  }

  vec4 star(float dist) {
    // x is 1 at the edge of the core; the sprite extends to x = halo.
    float x = dist * 2.0 * v_halo;
    float core = exp(-x * x * 0.7);
    float glow = pow(1.0 + x * x * 0.5, -1.4);
    float fade = 1.0 - smoothstep(v_halo * 0.35, v_halo, x);

    vec3 color = v_color;
    if (isSprite(SPRITE_ARMSTAR)) color = mix(color, WARM, 0.25 * glow);
    if (isSprite(SPRITE_NUCLEUS)) color = mix(color, WARM, 0.4 * glow);
    // Bright cores saturate to white, leaving the colour in the halo.
    if (u_light < 0.5) color = mix(color, vec3(1.0), core * 0.3);
    return vec4(color, v_alpha * (core * 0.8 + glow * 0.55) * fade);
  }

  // Gaussian glows: brightest at the centre and falling off continuously, so
  // no sprite has a flat top or a visible rim (which reads as a bokeh disc).
  vec4 glow(float dist, float peak) {
    return vec4(v_color, v_alpha * peak * exp(-dist * dist * 13.0));
  }

  void main() {
    float dist = length(gl_PointCoord - 0.5);
    if (dist > 0.5) discard;

    vec4 color;
    if (isSprite(SPRITE_DUSTLANE)) {
      color = glow(dist, 1.0);
      if (u_light < 0.5) {
        // The absorbing pass multiplies what is behind by (1 - rgb).
        gl_FragColor = vec4(DUST_ABSORPTION * color.a, color.a);
        return;
      }
    } else if (isSprite(SPRITE_CLOUD)) {
      color = glow(dist, 1.0);
    } else if (isSprite(SPRITE_GAS)) {
      color = glow(dist, 1.4);
    } else if (isSprite(SPRITE_DUST)) {
      color = glow(dist, 2.0);
    } else {
      color = star(dist);
    }
    if (color.a < 0.002) discard;

    if (u_light < 0.5) {
      color.rgb = mix(vec3(dot(color.rgb, LUMA)), color.rgb, SATURATION);
    }
    gl_FragColor = u_premultiply > 0.5 ? vec4(color.rgb * color.a, color.a) : color;
  }
`;

/** Draws a full-screen triangle that samples an offscreen layer. */
export const COMPOSITE_VERTEX_SHADER = /* glsl */ `
  attribute vec2 a_corner;
  varying vec2 v_uv;

  void main() {
    v_uv = a_corner * 0.5 + 0.5;
    gl_Position = vec4(a_corner, 0.0, 1.0);
  }
`;

export const COMPOSITE_FRAGMENT_SHADER = /* glsl */ `
  precision mediump float;
  uniform sampler2D u_layer;
  varying vec2 v_uv;

  void main() {
    gl_FragColor = texture2D(u_layer, v_uv);
  }
`;
