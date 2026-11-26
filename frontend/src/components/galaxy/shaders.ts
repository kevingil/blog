/** Positions arrive in canvas pixels (origin top-left) and become clip space. */
export const VERTEX_SHADER = /* glsl */ `
  attribute vec2 a_position;
  attribute float a_size;
  attribute vec3 a_color;
  attribute float a_alpha;
  attribute float a_spiralArm;

  uniform vec2 u_resolution;

  varying vec3 v_color;
  varying float v_alpha;
  varying float v_spiralArm;

  void main() {
    vec2 clip = (a_position / u_resolution) * 2.0 - 1.0;
    gl_Position = vec4(clip.x, -clip.y, 0.0, 1.0);
    gl_PointSize = a_size;
    v_color = a_color;
    v_alpha = a_alpha;
    v_spiralArm = a_spiralArm;
  }
`;

/**
 * Draws each point as a soft round sprite. The falloff profile depends on
 * what the point looks like it is, inferred from its colour and opacity.
 */
export const FRAGMENT_SHADER = /* glsl */ `
  precision mediump float;

  varying vec3 v_color;
  varying float v_alpha;
  varying float v_spiralArm;

  void main() {
    float dist = length(gl_PointCoord - 0.5);
    if (dist > 0.5) {
      discard;
    }
    float d2 = dist * dist;

    bool isGlowingGas = v_alpha > 0.15 && v_color.r > 0.7 && v_color.g > 0.7;
    bool isSpiralArm = v_spiralArm > 0.5;
    bool isCore = v_color.r > 0.95 && v_color.g > 0.8 && v_color.b < 0.7;
    bool isColorfulDust = v_alpha > 0.05 && v_alpha < 0.25 && (v_color.r > 0.4 || v_color.g > 0.3);

    float alpha;
    vec3 color = v_color;
    if (isGlowingGas) {
      // Very soft and wide, so overlapping gas blends smoothly.
      alpha = v_alpha * (exp(-d2 * 2.0) + exp(-d2 * 0.8) * 0.6 + exp(-d2 * 0.3) * 0.3);
    } else if (isSpiralArm || isCore) {
      // Bright centre with a warm halo.
      alpha = v_alpha * (exp(-d2 * 6.0) * 1.3 + exp(-d2 * 1.5) * 0.9 + exp(-d2 * 0.4) * 0.6);
      float warmth = isCore ? 0.4 : 0.25;
      color = mix(v_color, vec3(1.0, 0.85, 0.3), warmth * (1.0 - dist));
    } else if (isColorfulDust) {
      alpha = v_alpha * (exp(-d2 * 1.5) * 1.5 + exp(-d2 * 0.6) * 1.2 + exp(-d2 * 0.25) * 0.8);
    } else if (v_alpha < 0.3) {
      // Faint star.
      alpha = v_alpha * (exp(-d2 * 4.0) * 0.6 + exp(-d2 * 1.0) * 0.3);
    } else {
      // Ordinary star.
      alpha = v_alpha * (exp(-d2 * 5.0) + exp(-d2 * 2.0) * 0.4);
    }
    gl_FragColor = vec4(color, alpha);
  }
`;
