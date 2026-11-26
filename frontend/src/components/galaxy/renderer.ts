import { BACKGROUND, type ThemeMode } from './config';
import { FLOATS_PER_VERTEX } from './frame';
import { FRAGMENT_SHADER, VERTEX_SHADER } from './shaders';

const BYTES_PER_VERTEX = FLOATS_PER_VERTEX * Float32Array.BYTES_PER_ELEMENT;

/** Attribute name and float count, in interleaved order. */
const ATTRIBUTES: ReadonlyArray<[name: string, size: number]> = [
  ['a_position', 2],
  ['a_size', 1],
  ['a_color', 3],
  ['a_alpha', 1],
  ['a_spiralArm', 1],
];

/** Owns the WebGL program and vertex buffer, and draws prepared frames. */
export class GalaxyRenderer {
  private constructor(
    private readonly gl: WebGLRenderingContext,
    private readonly program: WebGLProgram,
    private readonly buffer: WebGLBuffer,
    private readonly resolution: WebGLUniformLocation | null,
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

    gl.enable(gl.BLEND);
    return new GalaxyRenderer(gl, program, buffer, gl.getUniformLocation(program, 'u_resolution'));
  }

  /** Clears to the theme background and draws the first `count` vertices of `vertices`. */
  draw(vertices: Float32Array, count: number, theme: ThemeMode): void {
    const { gl } = this;
    const { width, height } = gl.canvas;

    gl.viewport(0, 0, width, height);
    gl.clearColor(...BACKGROUND[theme], 1.0);
    // Dark mode adds light for a glow; light mode layers dark ink normally.
    if (theme === 'light') {
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
    } else {
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE);
    }
    gl.clear(gl.COLOR_BUFFER_BIT);
    if (count === 0) return;

    gl.uniform2f(this.resolution, width, height);
    gl.bufferSubData(gl.ARRAY_BUFFER, 0, vertices.subarray(0, count * FLOATS_PER_VERTEX));
    gl.drawArrays(gl.POINTS, 0, count);
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
