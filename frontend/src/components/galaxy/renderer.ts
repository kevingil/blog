import { BACKGROUND, type ThemeMode } from './config';
import { FLOATS_PER_VERTEX, type FrameCounts } from './frame';
import { FRAGMENT_SHADER, VERTEX_SHADER } from './shaders';

const BYTES_PER_VERTEX = FLOATS_PER_VERTEX * Float32Array.BYTES_PER_ELEMENT;

/** Attribute name and float count, in interleaved order. */
const ATTRIBUTES: ReadonlyArray<[name: string, size: number]> = [
  ['a_position', 2],
  ['a_size', 1],
  ['a_color', 3],
  ['a_alpha', 1],
  ['a_kind', 1],
  ['a_halo', 1],
];

const UNIFORMS = ['u_resolution', 'u_pixelRatio', 'u_maxPointSize', 'u_light'] as const;
type UniformName = (typeof UNIFORMS)[number];

export interface DrawParams {
  theme: ThemeMode;
  /** Viewport in CSS pixels; vertex positions use the same units. */
  cssWidth: number;
  cssHeight: number;
  pixelRatio: number;
}

/** Owns the WebGL program and vertex buffer, and draws prepared frames. */
export class GalaxyRenderer {
  private constructor(
    private readonly gl: WebGLRenderingContext,
    private readonly program: WebGLProgram,
    private readonly buffer: WebGLBuffer,
    private readonly uniforms: Record<UniformName, WebGLUniformLocation | null>,
    private readonly maxPointSize: number,
  ) {}

  /** Returns `null` when WebGL is unavailable or the shaders fail to build. */
  static create(canvas: HTMLCanvasElement, maxVertices: number): GalaxyRenderer | null {
    const gl = canvas.getContext('webgl', { alpha: true, premultipliedAlpha: false, antialias: true });
    if (!gl) {
      console.warn('Galaxy animation disabled: WebGL is not available.');
      return null;
    }

    const program = linkProgram(gl);
    const buffer = gl.createBuffer();
    if (!program || !buffer) return null;

    gl.useProgram(program);
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    gl.bufferData(gl.ARRAY_BUFFER, maxVertices * BYTES_PER_VERTEX, gl.DYNAMIC_DRAW);

    let offset = 0;
    for (const [name, size] of ATTRIBUTES) {
      const location = gl.getAttribLocation(program, name);
      gl.enableVertexAttribArray(location);
      gl.vertexAttribPointer(location, size, gl.FLOAT, false, BYTES_PER_VERTEX, offset);
      offset += size * Float32Array.BYTES_PER_ELEMENT;
    }

    const uniforms = Object.fromEntries(
      UNIFORMS.map((name) => [name, gl.getUniformLocation(program, name)]),
    ) as Record<UniformName, WebGLUniformLocation | null>;
    const [, maxPointSize] = gl.getParameter(gl.ALIASED_POINT_SIZE_RANGE) as Float32Array;

    gl.enable(gl.BLEND);
    return new GalaxyRenderer(gl, program, buffer, uniforms, maxPointSize);
  }

  /**
   * Clears to the theme background, draws the emissive vertices, then the
   * absorbing ones (dust lanes) on top.
   */
  draw(vertices: Float32Array, counts: FrameCounts, params: DrawParams): void {
    const { gl, uniforms } = this;
    const light = params.theme === 'light';

    gl.viewport(0, 0, gl.drawingBufferWidth, gl.drawingBufferHeight);
    gl.clearColor(...BACKGROUND[params.theme], 1.0);
    // Dark mode adds light for a glow; light mode layers dark ink normally.
    if (light) {
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
    } else {
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE);
    }
    gl.clear(gl.COLOR_BUFFER_BIT);
    if (counts.total === 0) return;

    gl.uniform2f(uniforms.u_resolution, params.cssWidth, params.cssHeight);
    gl.uniform1f(uniforms.u_pixelRatio, params.pixelRatio);
    gl.uniform1f(uniforms.u_maxPointSize, this.maxPointSize);
    gl.uniform1f(uniforms.u_light, light ? 1 : 0);
    gl.bufferSubData(gl.ARRAY_BUFFER, 0, vertices.subarray(0, counts.total * FLOATS_PER_VERTEX));
    gl.drawArrays(gl.POINTS, 0, counts.emissive);

    const absorbing = counts.total - counts.emissive;
    if (absorbing > 0) {
      // Dark mode: dust removes light (dst × (1 − a)). Light mode: it is more ink.
      if (!light) gl.blendFunc(gl.ZERO, gl.ONE_MINUS_SRC_ALPHA);
      gl.drawArrays(gl.POINTS, counts.emissive, absorbing);
    }
  }

  /** Frees GPU resources. Leaves the context itself alive so the canvas can be reused. */
  dispose(): void {
    if (this.gl.isContextLost()) return;
    this.gl.deleteBuffer(this.buffer);
    this.gl.deleteProgram(this.program);
  }
}

function linkProgram(gl: WebGLRenderingContext): WebGLProgram | null {
  const vertex = compileShader(gl, gl.VERTEX_SHADER, VERTEX_SHADER);
  const fragment = compileShader(gl, gl.FRAGMENT_SHADER, FRAGMENT_SHADER);
  const program = gl.createProgram();
  if (!vertex || !fragment || !program) return null;

  gl.attachShader(program, vertex);
  gl.attachShader(program, fragment);
  gl.linkProgram(program);
  // The program keeps its own reference; the shader objects are no longer needed.
  gl.deleteShader(vertex);
  gl.deleteShader(fragment);

  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    console.error('Galaxy shader program failed to link:', gl.getProgramInfoLog(program));
    gl.deleteProgram(program);
    return null;
  }
  return program;
}

function compileShader(gl: WebGLRenderingContext, type: number, source: string): WebGLShader | null {
  const shader = gl.createShader(type);
  if (!shader) return null;

  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    console.error('Galaxy shader failed to compile:', gl.getShaderInfoLog(shader));
    gl.deleteShader(shader);
    return null;
  }
  return shader;
}
