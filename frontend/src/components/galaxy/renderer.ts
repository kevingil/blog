import { BACKGROUND, SOFT_LAYER_SCALE, type ThemeMode } from './config';
import { FLOATS_PER_VERTEX, type FrameCounts } from './frame';
import { COMPOSITE_FRAGMENT_SHADER, COMPOSITE_VERTEX_SHADER, FRAGMENT_SHADER, VERTEX_SHADER } from './shaders';

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

const UNIFORMS = ['u_resolution', 'u_pixelRatio', 'u_maxPointSize', 'u_light', 'u_premultiply'] as const;
type UniformName = (typeof UNIFORMS)[number];

export interface DrawParams {
  theme: ThemeMode;
  /** Viewport in CSS pixels; vertex positions use the same units. */
  cssWidth: number;
  cssHeight: number;
  pixelRatio: number;
}

/** An offscreen colour target that is later stretched over the screen. */
interface Layer {
  framebuffer: WebGLFramebuffer;
  texture: WebGLTexture;
}

/**
 * Owns the WebGL programs and buffers, and draws prepared frames.
 *
 * Gas and dust are smooth, so they are drawn into two low-resolution layers
 * (`SOFT_LAYER_SCALE` of CSS resolution) and stretched over the screen; they
 * cover far more pixels than anything else, so this is most of the GPU
 * saving. Stars stay at full resolution. Per frame:
 * 1. gas → glow layer; dust lanes → dust layer;
 * 2. screen: background, glow layer, stars, then the dust layer on top.
 */
export class GalaxyRenderer {
  private layerWidth = 0;
  private layerHeight = 0;

  private constructor(
    private readonly gl: WebGLRenderingContext,
    private readonly program: WebGLProgram,
    private readonly buffer: WebGLBuffer,
    private readonly uniforms: Record<UniformName, WebGLUniformLocation | null>,
    private readonly maxPointSize: number,
    private readonly composite: WebGLProgram,
    private readonly quad: WebGLBuffer,
    private readonly glowLayer: Layer,
    private readonly dustLayer: Layer,
  ) {}

  /** Returns `null` when WebGL is unavailable or the shaders fail to build. */
  static create(canvas: HTMLCanvasElement, maxVertices: number): GalaxyRenderer | null {
    const gl = canvas.getContext('webgl', { alpha: true, premultipliedAlpha: false, antialias: true });
    if (!gl) {
      console.warn('Galaxy animation disabled: WebGL is not available.');
      return null;
    }

    const program = linkProgram(gl, VERTEX_SHADER, FRAGMENT_SHADER);
    const composite = linkProgram(gl, COMPOSITE_VERTEX_SHADER, COMPOSITE_FRAGMENT_SHADER);
    const buffer = gl.createBuffer();
    const quad = gl.createBuffer();
    const glowLayer = createLayer(gl);
    const dustLayer = createLayer(gl);
    if (!program || !composite || !buffer || !quad || !glowLayer || !dustLayer) return null;

    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    gl.bufferData(gl.ARRAY_BUFFER, maxVertices * BYTES_PER_VERTEX, gl.DYNAMIC_DRAW);
    // One triangle that covers the whole screen.
    gl.bindBuffer(gl.ARRAY_BUFFER, quad);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);

    const uniforms = Object.fromEntries(
      UNIFORMS.map((name) => [name, gl.getUniformLocation(program, name)]),
    ) as Record<UniformName, WebGLUniformLocation | null>;
    const [, maxPointSize] = gl.getParameter(gl.ALIASED_POINT_SIZE_RANGE) as Float32Array;

    gl.enable(gl.BLEND);
    return new GalaxyRenderer(gl, program, buffer, uniforms, maxPointSize, composite, quad, glowLayer, dustLayer);
  }

  draw(vertices: Float32Array, counts: FrameCounts, params: DrawParams): void {
    const { gl } = this;
    const light = params.theme === 'light';
    this.fitLayers(params);

    if (counts.total > 0) {
      gl.useProgram(this.program);
      this.bindPointAttributes();
      gl.bufferSubData(gl.ARRAY_BUFFER, 0, vertices.subarray(0, counts.total * FLOATS_PER_VERTEX));
      this.setPointUniforms(params, light);

      // Low-resolution layers. Light mode accumulates premultiplied ink so the
      // layer can be laid over the page with ordinary alpha blending.
      gl.uniform1f(this.uniforms.u_pixelRatio, SOFT_LAYER_SCALE);
      gl.uniform1f(this.uniforms.u_premultiply, light ? 1 : 0);
      this.beginLayer(this.glowLayer);
      if (light) gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA);
      else gl.blendFunc(gl.SRC_ALPHA, gl.ONE);
      gl.drawArrays(gl.POINTS, 0, counts.soft);

      this.beginLayer(this.dustLayer);
      // Dark mode: each puff writes how much light it blocks per channel, and
      // overlapping puffs combine as 1 − Π(1 − a).
      if (light) gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA);
      else gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_COLOR);
      gl.drawArrays(gl.POINTS, counts.emissive, counts.total - counts.emissive);
    }

    // The screen.
    gl.bindFramebuffer(gl.FRAMEBUFFER, null);
    gl.viewport(0, 0, gl.drawingBufferWidth, gl.drawingBufferHeight);
    gl.clearColor(...BACKGROUND[params.theme], 1.0);
    gl.clear(gl.COLOR_BUFFER_BIT);
    if (counts.total === 0) return;

    // Dark mode adds light; light mode lays ink over the page.
    this.drawLayer(this.glowLayer, gl.ONE, light ? gl.ONE_MINUS_SRC_ALPHA : gl.ONE);

    gl.useProgram(this.program);
    this.bindPointAttributes();
    gl.uniform1f(this.uniforms.u_pixelRatio, params.pixelRatio);
    gl.uniform1f(this.uniforms.u_premultiply, 0);
    gl.blendFuncSeparate(gl.SRC_ALPHA, light ? gl.ONE_MINUS_SRC_ALPHA : gl.ONE, gl.ZERO, gl.ONE);
    gl.drawArrays(gl.POINTS, counts.soft, counts.emissive - counts.soft);

    // Dark mode: dust removes light per channel, more blue than red, so lanes
    // look reddish-brown. Light mode: it is more ink.
    this.drawLayer(this.dustLayer, light ? gl.ONE : gl.ZERO, light ? gl.ONE_MINUS_SRC_ALPHA : gl.ONE_MINUS_SRC_COLOR);
  }

  /** Frees GPU resources. Leaves the context itself alive so the canvas can be reused. */
  dispose(): void {
    const { gl } = this;
    if (gl.isContextLost()) return;
    gl.deleteBuffer(this.buffer);
    gl.deleteBuffer(this.quad);
    gl.deleteProgram(this.program);
    gl.deleteProgram(this.composite);
    for (const layer of [this.glowLayer, this.dustLayer]) {
      gl.deleteFramebuffer(layer.framebuffer);
      gl.deleteTexture(layer.texture);
    }
  }

  /** Resizes the layer textures when the viewport changes. */
  private fitLayers(params: DrawParams): void {
    const width = Math.max(1, Math.ceil(params.cssWidth * SOFT_LAYER_SCALE));
    const height = Math.max(1, Math.ceil(params.cssHeight * SOFT_LAYER_SCALE));
    if (width === this.layerWidth && height === this.layerHeight) return;

    const { gl } = this;
    for (const layer of [this.glowLayer, this.dustLayer]) {
      gl.bindTexture(gl.TEXTURE_2D, layer.texture);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, null);
    }
    this.layerWidth = width;
    this.layerHeight = height;
  }

  private beginLayer(layer: Layer): void {
    const { gl } = this;
    gl.bindFramebuffer(gl.FRAMEBUFFER, layer.framebuffer);
    gl.viewport(0, 0, this.layerWidth, this.layerHeight);
    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT);
  }

  /**
   * Stretches a layer over the screen with the given colour blend factors.
   * The canvas's own alpha is left opaque, so the page never shows through.
   */
  private drawLayer(layer: Layer, source: number, destination: number): void {
    const { gl } = this;
    gl.useProgram(this.composite);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.quad);
    const location = gl.getAttribLocation(this.composite, 'a_corner');
    gl.enableVertexAttribArray(location);
    gl.vertexAttribPointer(location, 2, gl.FLOAT, false, 0, 0);
    gl.activeTexture(gl.TEXTURE0);
    gl.bindTexture(gl.TEXTURE_2D, layer.texture);
    gl.uniform1i(gl.getUniformLocation(this.composite, 'u_layer'), 0);
    gl.blendFuncSeparate(source, destination, gl.ZERO, gl.ONE);
    gl.drawArrays(gl.TRIANGLES, 0, 3);
  }

  /** The composite pass repoints attribute 0, so the point attributes are rebound before each point draw. */
  private bindPointAttributes(): void {
    const { gl } = this;
    gl.bindBuffer(gl.ARRAY_BUFFER, this.buffer);
    let offset = 0;
    for (const [name, size] of ATTRIBUTES) {
      const location = gl.getAttribLocation(this.program, name);
      gl.enableVertexAttribArray(location);
      gl.vertexAttribPointer(location, size, gl.FLOAT, false, BYTES_PER_VERTEX, offset);
      offset += size * Float32Array.BYTES_PER_ELEMENT;
    }
  }

  private setPointUniforms(params: DrawParams, light: boolean): void {
    const { gl, uniforms } = this;
    gl.uniform2f(uniforms.u_resolution, params.cssWidth, params.cssHeight);
    gl.uniform1f(uniforms.u_maxPointSize, this.maxPointSize);
    gl.uniform1f(uniforms.u_light, light ? 1 : 0);
  }
}

function createLayer(gl: WebGLRenderingContext): Layer | null {
  const texture = gl.createTexture();
  const framebuffer = gl.createFramebuffer();
  if (!texture || !framebuffer) return null;

  gl.bindTexture(gl.TEXTURE_2D, texture);
  // Linear filtering does the smooth upscale; 1×1 until the first resize.
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
  gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, null);
  gl.bindFramebuffer(gl.FRAMEBUFFER, framebuffer);
  gl.framebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture, 0);
  gl.bindFramebuffer(gl.FRAMEBUFFER, null);
  return { texture, framebuffer };
}

function linkProgram(gl: WebGLRenderingContext, vertexSource: string, fragmentSource: string): WebGLProgram | null {
  const vertex = compileShader(gl, gl.VERTEX_SHADER, vertexSource);
  const fragment = compileShader(gl, gl.FRAGMENT_SHADER, fragmentSource);
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
