import React, { useEffect, useRef, useState } from 'react';

interface StarParticle {
  x: number;
  y: number;
  z: number;
  brightness: number;
  color: string;
  size: number;
  temperature: number; // For realistic stellar colors
  armIndex?: number;
  isCore?: boolean;
  isDust?: boolean;
  isHII?: boolean;
}

interface GalaxyParams {
  coreRadius: number;
  diskRadius: number;
  armPitch: number;
  inclination: number;
  orientation: number;
  particleCount: number;
  rotationSpeed: number;
  timeScale: number;
}

// WebGL shader sources
const vertexShaderSource = `
  attribute vec2 a_position;
  attribute float a_size;
  attribute vec3 a_color;
  attribute float a_brightness;
  
  uniform mat3 u_transform;
  uniform vec2 u_resolution;
  
  varying vec3 v_color;
  varying float v_brightness;
  
  void main() {
    vec3 transformed = u_transform * vec3(a_position, 1.0);
    vec2 position = transformed.xy;
    
    gl_Position = vec4(
      (position / u_resolution) * 2.0 - 1.0,
      0.0,
      1.0
    );
    gl_Position.y *= -1.0;
    
    gl_PointSize = a_size;
    v_color = a_color;
    v_brightness = a_brightness;
  }
`;

const fragmentShaderSource = `
  precision mediump float;
  
  varying vec3 v_color;
  varying float v_brightness;
  
  void main() {
    vec2 center = gl_PointCoord - 0.5;
    float dist = length(center);
    
    if (dist > 0.5) {
      discard;
    }
    
    // Determine if this is a nebula particle (very dim) or star
    bool isNebula = v_brightness < 0.25;
    
    float alpha;
    if (isNebula) {
      // Softer, more diffuse falloff for nebula/dust
      alpha = exp(-dist * dist * 2.0) * v_brightness;
      
      // Add very soft glow for smoke effect
      float softGlow = exp(-dist * dist * 0.5) * v_brightness * 0.8;
      alpha += softGlow;
    } else {
      // Sharp, bright falloff for stars
      alpha = exp(-dist * dist * 8.0) * v_brightness;
      
      // Add stellar glow
      float glow = exp(-dist * dist * 2.0) * v_brightness * 0.3;
      alpha += glow;
    }
    
    gl_FragColor = vec4(v_color, alpha);
  }
`;

// Create and compile shader
function createShader(gl: WebGLRenderingContext, type: number, source: string): WebGLShader | null {
  const shader = gl.createShader(type);
  if (!shader) return null;
  
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    console.error('Shader compilation error:', gl.getShaderInfoLog(shader));
    gl.deleteShader(shader);
    return null;
  }
  
  return shader;
}

// Create shader program
function createProgram(gl: WebGLRenderingContext, vertexShader: WebGLShader, fragmentShader: WebGLShader): WebGLProgram | null {
  const program = gl.createProgram();
  if (!program) return null;
  
  gl.attachShader(program, vertexShader);
  gl.attachShader(program, fragmentShader);
  gl.linkProgram(program);
  
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    console.error('Program linking error:', gl.getProgramInfoLog(program));
    gl.deleteProgram(program);
    return null;
  }
  
  return program;
}

// Convert temperature to RGB color (simplified blackbody radiation)
function temperatureToColor(temp: number): [number, number, number] {
  // Temperature in Kelvin, typical range: 3000K (red) to 30000K (blue)
  temp = Math.max(1000, Math.min(40000, temp));
  
  let r, g, b;
  
  if (temp < 3700) {
    r = 1.0;
    g = Math.max(0, (temp - 1000) / 2700);
    b = 0.0;
  } else if (temp < 5500) {
    r = 1.0;
    g = 0.39 + 0.61 * (temp - 3700) / 1800;
    b = Math.max(0, (temp - 3700) / 1800);
  } else if (temp < 7500) {
    r = 1.0 - 0.2 * (temp - 5500) / 2000;
    g = 1.0;
    b = 0.6 + 0.4 * (temp - 5500) / 2000;
  } else {
    r = 0.7 - 0.3 * Math.min(1, (temp - 7500) / 10000);
    g = 0.8 + 0.2 * Math.min(1, (temp - 7500) / 10000);
    b = 1.0;
  }
  
  return [r, g, b];
}

export const SpiralGalaxyAnimation: React.FC<{ zIndex?: number }> = ({ zIndex = -1 }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animationRef = useRef<number>();
  const glRef = useRef<WebGLRenderingContext | null>(null);
  const programRef = useRef<WebGLProgram | null>(null);
  const buffersRef = useRef<{
    position: WebGLBuffer | null;
    size: WebGLBuffer | null;
    color: WebGLBuffer | null;
    brightness: WebGLBuffer | null;
  }>({
    position: null,
    size: null,
    color: null,
    brightness: null
  });
  const [isPlaying] = useState(true);
  const timeSpeed = 3; // Fixed 3x speed
  const zoom = 1.2; // Fixed zoom level
  const [rotation, setRotation] = useState(0);
  const timeRef = useRef(0);

  // Galaxy parameters based on Andromeda M31
  const galaxyParams: GalaxyParams = {
    coreRadius: 50, // Scaled for viewport
    diskRadius: 350, // Slightly smaller main disk for better spiral definition
    armPitch: Math.PI / 18, // Slightly tighter spiral arms
    inclination: 80 * Math.PI / 180, // 87° from face-on (more tilted/flatter)
    orientation: 20 * Math.PI / 180, // Position angle
    particleCount: 5000, // Adjusted for extended distribution
    rotationSpeed: 0.00001,
    timeScale: 10000000 // 1 sim second = 10 million years
  };

  // Generate spiral galaxy structure with natural distribution and realistic colors
  const generateGalaxyParticles = (): StarParticle[] => {
    const particles: StarParticle[] = [];
    
    // Increase particle count to account for extended distribution
    const totalParticles = galaxyParams.particleCount * 1.5;
    
    // Add Active Galactic Nucleus (AGN) - very bright central object
    particles.push({
      x: 0,
      y: 0,
      z: 0,
      brightness: 1.0,
      color: '#FFFFFF',
      size: 8.0,
      temperature: 20000, // Very hot, blue-white
      isCore: true
    });

    // Generate nebula/dust particles for smoke effect
    const nebulaParticles = Math.floor(totalParticles * 0.8); // 80% more particles for nebula
    for (let i = 0; i < nebulaParticles; i++) {
      const u = Math.random();
      const v = Math.random();
      
      // Focus nebula in disk regions
      let r: number;
      if (u < 0.9) {
        r = galaxyParams.coreRadius + (galaxyParams.diskRadius * 1.2 - galaxyParams.coreRadius) * Math.sqrt(v);
      } else {
        r = galaxyParams.diskRadius * 1.2 + (galaxyParams.diskRadius * 0.5) * Math.pow(v, 2);
      }
      
      const baseTheta = Math.random() * 2 * Math.PI;
      
      // Nebula follows spiral structure but is more diffuse
      const spiralFactor = 2;
      const spiralAngle = spiralFactor * (baseTheta - galaxyParams.armPitch * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
      const armAngle = spiralAngle % (2 * Math.PI / spiralFactor);
      const distanceToArm = Math.min(armAngle, (2 * Math.PI / spiralFactor) - armAngle);
      
      // Wider nebula arms
      const nebulaArmWidth = Math.PI / 4;
      const armStrength = Math.exp(-Math.pow(distanceToArm / nebulaArmWidth, 2));
      
      // Nebula density threshold - more permissive
      const nebulaThreshold = 0.6 + armStrength * 0.3;
      if (Math.random() > nebulaThreshold) continue;
      
      const perturbation = (Math.random() - 0.5) * 30;
      const theta = baseTheta + perturbation / Math.max(r, 10);
      
      const x = r * Math.cos(theta);
      const y = r * Math.sin(theta);
      
      // Thicker vertical distribution for nebula
      const nebulaHeight = 25 * Math.exp(-r / (galaxyParams.diskRadius * 0.8));
      const z = (Math.random() - 0.5) * Math.max(nebulaHeight, 5);
      
      // Nebula colors - dust and gas
      let nebulaColor: string;
      let temperature: number;
      
      if (armStrength > 0.3 && Math.random() < 0.4) {
        // HII regions - pinkish-red emission nebulae
        nebulaColor = `rgb(${255}, ${Math.floor(100 + Math.random() * 100)}, ${Math.floor(120 + Math.random() * 80)})`;
        temperature = 10000; // Hot ionized gas
      } else if (Math.random() < 0.3) {
        // Blue reflection nebulae
        nebulaColor = `rgb(${Math.floor(100 + Math.random() * 100)}, ${Math.floor(150 + Math.random() * 80)}, ${255})`;
        temperature = 7000;
      } else {
        // Dark nebulae and dust - brownish/reddish
        const dustRed = Math.floor(80 + Math.random() * 60);
        const dustGreen = Math.floor(40 + Math.random() * 40);
        const dustBlue = Math.floor(20 + Math.random() * 30);
        nebulaColor = `rgb(${dustRed}, ${dustGreen}, ${dustBlue})`;
        temperature = 100; // Cold dust
      }
      
      const brightness = 0.08 + Math.random() * 0.12; // Much dimmer than stars
      const size = 2.0 + Math.random() * 4.0; // Larger, more diffuse
      
      particles.push({
        x,
        y,
        z,
        brightness,
        color: nebulaColor,
        size,
        temperature,
        isDust: true
      });
         }
    
    for (let i = 0; i < totalParticles; i++) {
      // Generate radius with more realistic distribution
      // Most particles in inner regions, but some extend well beyond visible disk
      const u = Math.random();
      const v = Math.random();
      
      // Create a more gradual falloff - some particles can be up to 2x the disk radius
      const maxRadius = galaxyParams.diskRadius * 2;
      let r: number;
      
      if (u < 0.7) {
        // 70% of particles in main disk with exponential distribution
        r = galaxyParams.coreRadius + (galaxyParams.diskRadius - galaxyParams.coreRadius) * Math.sqrt(v);
      } else {
        // 30% of particles in extended halo/outer regions
        const haloFactor = Math.pow(v, 3); // Cubic falloff for very sparse outer regions
        r = galaxyParams.diskRadius + (maxRadius - galaxyParams.diskRadius) * haloFactor;
      }
      
      // Random angle for base distribution
      const baseTheta = Math.random() * 2 * Math.PI;
      
      // Create realistic spiral arm structure
      const spiralFactor = 2; // Number of main spiral arms
      const spiralAngle = spiralFactor * (baseTheta - galaxyParams.armPitch * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
      
      // Calculate distance to nearest spiral arm
      const armAngle = spiralAngle % (2 * Math.PI / spiralFactor);
      const distanceToArm = Math.min(armAngle, (2 * Math.PI / spiralFactor) - armAngle);
      
      // Spiral arm width varies with radius (wider arms toward center)
      const baseArmWidth = Math.PI / 8; // Base arm width
      const armWidthVariation = 1 + 0.5 * Math.exp(-r / (galaxyParams.diskRadius * 0.3)); // Wider in center
      const effectiveArmWidth = baseArmWidth * armWidthVariation;
      
      // Primary spiral arms - strong density enhancement
      const armStrength = Math.exp(-Math.pow(distanceToArm / effectiveArmWidth, 2)); // Gaussian profile
      
      // Secondary spiral features (inter-arm spurs and structure)
      const spurAngle = spiralFactor * 1.5 * (baseTheta - galaxyParams.armPitch * 0.7 * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
      const spurDistance = Math.min(spurAngle % (2 * Math.PI / (spiralFactor * 1.5)), (2 * Math.PI / (spiralFactor * 1.5)) - (spurAngle % (2 * Math.PI / (spiralFactor * 1.5))));
      const spurStrength = 0.3 * Math.exp(-Math.pow(spurDistance / (effectiveArmWidth * 0.6), 2));
      
      // Combine spiral features
      const spiralStrength = Math.exp(-r / (galaxyParams.diskRadius * 1.2)); // Spiral structure fades with distance
      const combinedSpiral = (armStrength + spurStrength) * spiralStrength;
      
      // Create density threshold that varies with radius and spiral position
      // More even distribution like real Andromeda
      let densityThreshold: number;
      if (r <= galaxyParams.diskRadius) {
        // Main disk - moderate spiral enhancement with higher base density
        const baseDensity = 0.35; // Higher base for more even distribution
        const spiralBoost = combinedSpiral * 0.4; // Moderate spiral enhancement
        densityThreshold = baseDensity + spiralBoost;
      } else {
        // Extended regions - much sparser, weaker spiral structure
        const extendedFactor = Math.exp(-(r - galaxyParams.diskRadius) / (galaxyParams.diskRadius * 0.4));
        const baseDensity = 0.15 * extendedFactor; // Higher base for halo
        const spiralBoost = combinedSpiral * 0.2 * extendedFactor;
        densityThreshold = baseDensity + spiralBoost;
      }
      
      // Apply density threshold
      if (Math.random() > densityThreshold) continue;
      
      // Add perturbation that's slightly biased toward spiral arms for more natural clustering
      const perturbationScale = Math.min(20, r * 0.05);
      const randomPerturbation = (Math.random() - 0.5) * perturbationScale;
      
      // Slight bias toward spiral arm centers
      const armBias = armStrength > 0.1 ? (Math.random() - 0.5) * perturbationScale * 0.3 * armStrength : 0;
      const totalPerturbation = randomPerturbation + armBias;
      
      const theta = baseTheta + totalPerturbation / Math.max(r, 10);
      
      const x = r * Math.cos(theta);
      const y = r * Math.sin(theta);
      
      // Vertical distribution - gets thicker toward center, thinner at edges
      const baseScaleHeight = 15;
      const scaleHeight = baseScaleHeight * Math.exp(-r / (galaxyParams.diskRadius * 0.6));
      const z = (Math.random() - 0.5) * Math.max(scaleHeight, 2); // Minimum thickness
      
      // Realistic stellar populations and colors based on galactic position
      let temperature: number;
      let brightness: number;
      
      if (r <= galaxyParams.coreRadius * 2) {
        // Core region - old, red giant stars and blue supergiants
        if (Math.random() < 0.8) {
          temperature = 3000 + Math.random() * 2000; // Red giants (K-M class)
          brightness = 0.6 + Math.random() * 0.4;
        } else {
          temperature = 15000 + Math.random() * 15000; // Blue supergiants (O-B class)
          brightness = 0.8 + Math.random() * 0.2;
        }
      } else if (r <= galaxyParams.diskRadius && armStrength > 0.3) {
        // Spiral arms - young, hot blue stars and star-forming regions
        if (Math.random() < 0.6) {
          temperature = 8000 + Math.random() * 12000; // Hot blue-white stars (A-B class)
          brightness = 0.7 + Math.random() * 0.3;
        } else {
          temperature = 4000 + Math.random() * 3000; // Sun-like stars (G-K class)
          brightness = 0.4 + Math.random() * 0.4;
        }
      } else if (r <= galaxyParams.diskRadius) {
        // Inter-arm regions - older, cooler stars
        temperature = 3500 + Math.random() * 3500; // Cool stars (K-G class)
        brightness = 0.3 + Math.random() * 0.4;
      } else {
        // Halo - old, metal-poor stars
        temperature = 3000 + Math.random() * 2500; // Old red stars
        const haloDistance = r - galaxyParams.diskRadius;
        brightness = Math.exp(-haloDistance / (galaxyParams.diskRadius * 0.3)) * (0.05 + Math.random() * 0.15);
      }
      
      // Apply general brightness falloff
      if (r <= galaxyParams.diskRadius) {
        brightness *= Math.exp(-r / (galaxyParams.diskRadius * 0.5));
      }
      
      // Size variation based on brightness and temperature
      const baseSize = 0.3 + Math.random() * 1.2;
      const tempFactor = Math.max(0.5, Math.min(2.0, temperature / 6000)); // Hotter stars appear larger
      const size = baseSize * (0.5 + brightness * 1.5) * tempFactor;
      
      const [r_color, g_color, b_color] = temperatureToColor(temperature);
      const color = `rgb(${Math.round(r_color * 255)}, ${Math.round(g_color * 255)}, ${Math.round(b_color * 255)})`;
      
      particles.push({
        x,
        y,
        z,
        brightness: Math.min(brightness, 1),
        color,
        size: Math.max(size, 0.2),
        temperature,
      });
    }

    return particles;
  };

  // Initialize WebGL
  const initWebGL = (): boolean => {
    const canvas = canvasRef.current;
    if (!canvas) return false;

    const gl = canvas.getContext('webgl', {
      alpha: true,
      premultipliedAlpha: false,
      antialias: true
    });
    
    if (!gl) {
      console.error('WebGL not supported');
      return false;
    }

    glRef.current = gl;

    // Create shaders
    const vertexShader = createShader(gl, gl.VERTEX_SHADER, vertexShaderSource);
    const fragmentShader = createShader(gl, gl.FRAGMENT_SHADER, fragmentShaderSource);

    if (!vertexShader || !fragmentShader) return false;

    // Create program
    const program = createProgram(gl, vertexShader, fragmentShader);
    if (!program) return false;

    programRef.current = program;

    // Create buffers
    buffersRef.current = {
      position: gl.createBuffer(),
      size: gl.createBuffer(),
      color: gl.createBuffer(),
      brightness: gl.createBuffer()
    };

    // Enable blending for transparency
    gl.enable(gl.BLEND);
    gl.blendFunc(gl.SRC_ALPHA, gl.ONE); // Additive blending for glow effect

    return true;
  };

  // Apply 3D rotation and projection
  const projectParticle = (particle: StarParticle, time: number, canvasWidth: number, canvasHeight: number) => {
    // Apply galaxy rotation (differential rotation)
    const r = Math.sqrt(particle.x * particle.x + particle.y * particle.y);
    let rotationRate = galaxyParams.rotationSpeed;
    
    if (r > galaxyParams.coreRadius) {
      // Flat rotation curve in outer regions
      rotationRate = galaxyParams.rotationSpeed * galaxyParams.coreRadius / r;
    }
    
    const currentAngle = Math.atan2(particle.y, particle.x);
    const newAngle = currentAngle + rotationRate * time * timeSpeed;
    
    const rotatedX = r * Math.cos(newAngle);
    const rotatedY = r * Math.sin(newAngle);
    const rotatedZ = particle.z;

    // Apply inclination (77° from face-on, flipped)
    const inclinedY = rotatedY * Math.cos(-galaxyParams.inclination) - rotatedZ * Math.sin(-galaxyParams.inclination);
    const inclinedZ = rotatedY * Math.sin(-galaxyParams.inclination) + rotatedZ * Math.cos(-galaxyParams.inclination);

    // Apply orientation (35° position angle)
    const orientedX = rotatedX * Math.cos(galaxyParams.orientation + rotation) - inclinedY * Math.sin(galaxyParams.orientation + rotation);
    const orientedY = rotatedX * Math.sin(galaxyParams.orientation + rotation) + inclinedY * Math.cos(galaxyParams.orientation + rotation);

    // Perspective projection
    const cameraDistance = 10000; // Camera distance from galaxy center
    const scale = cameraDistance / (cameraDistance + inclinedZ) * zoom;
    
    const screenX = canvasWidth / 2 - orientedX * scale + 175; // Mirror X axis + move camera 100px left
    const screenY = canvasHeight / 2 + orientedY * scale - 100;
    
    return {
      x: screenX,
      y: screenY,
      z: inclinedZ,
      scale,
      visible: screenX >= -50 && screenX <= canvasWidth + 50 && screenY >= -50 && screenY <= canvasHeight + 50
    };
  };

  const [particles] = useState(() => generateGalaxyParticles());

  const animate = () => {
    const canvas = canvasRef.current;
    const gl = glRef.current;
    const program = programRef.current;
    
    if (!canvas || !gl || !program) return;

    const { width, height } = canvas;
    
    gl.viewport(0, 0, width, height);
    gl.clearColor(0.02, 0.02, 0.06, 1.0); // Dark space background
    gl.clear(gl.COLOR_BUFFER_BIT);

    if (isPlaying) {
      timeRef.current += 16; // ~60fps
    }

    // Project particles
    const projectedParticles = particles
      .map(particle => ({
        particle,
        projected: projectParticle(particle, timeRef.current, width, height)
      }))
      .filter(p => p.projected.visible);

    if (projectedParticles.length === 0) return;

    // Prepare data arrays
    const positions: number[] = [];
    const sizes: number[] = [];
    const colors: number[] = [];
    const brightnesses: number[] = [];

    projectedParticles.forEach(({ particle, projected }) => {
      positions.push(projected.x, projected.y);
      
      const size = particle.size * projected.scale;
      sizes.push(Math.max(1.0, size));
      
      // Parse color
      const colorMatch = particle.color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
      if (colorMatch) {
        colors.push(
          parseInt(colorMatch[1]) / 255,
          parseInt(colorMatch[2]) / 255,
          parseInt(colorMatch[3]) / 255
        );
      } else {
        colors.push(1, 1, 1); // Default white
      }
      
      const alpha = particle.brightness * Math.min(1, projected.scale);
      brightnesses.push(alpha);
    });

    // Upload data to GPU
    gl.useProgram(program);

    // Position buffer
    gl.bindBuffer(gl.ARRAY_BUFFER, buffersRef.current.position);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(positions), gl.DYNAMIC_DRAW);
    const positionLocation = gl.getAttribLocation(program, 'a_position');
    gl.enableVertexAttribArray(positionLocation);
    gl.vertexAttribPointer(positionLocation, 2, gl.FLOAT, false, 0, 0);

    // Size buffer
    gl.bindBuffer(gl.ARRAY_BUFFER, buffersRef.current.size);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(sizes), gl.DYNAMIC_DRAW);
    const sizeLocation = gl.getAttribLocation(program, 'a_size');
    gl.enableVertexAttribArray(sizeLocation);
    gl.vertexAttribPointer(sizeLocation, 1, gl.FLOAT, false, 0, 0);

    // Color buffer
    gl.bindBuffer(gl.ARRAY_BUFFER, buffersRef.current.color);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(colors), gl.DYNAMIC_DRAW);
    const colorLocation = gl.getAttribLocation(program, 'a_color');
    gl.enableVertexAttribArray(colorLocation);
    gl.vertexAttribPointer(colorLocation, 3, gl.FLOAT, false, 0, 0);

    // Brightness buffer
    gl.bindBuffer(gl.ARRAY_BUFFER, buffersRef.current.brightness);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(brightnesses), gl.DYNAMIC_DRAW);
    const brightnessLocation = gl.getAttribLocation(program, 'a_brightness');
    gl.enableVertexAttribArray(brightnessLocation);
    gl.vertexAttribPointer(brightnessLocation, 1, gl.FLOAT, false, 0, 0);

    // Set uniforms
    const transformLocation = gl.getUniformLocation(program, 'u_transform');
    const resolutionLocation = gl.getUniformLocation(program, 'u_resolution');
    
    // Identity transform for now
    gl.uniformMatrix3fv(transformLocation, false, [
      1, 0, 0,
      0, 1, 0,
      0, 0, 1
    ]);
    
    gl.uniform2f(resolutionLocation, width, height);

    // Draw
    gl.drawArrays(gl.POINTS, 0, positions.length / 2);

    animationRef.current = requestAnimationFrame(animate);
  };

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const resizeCanvas = () => {
      canvas.width = window.innerWidth;
      canvas.height = window.innerHeight;
    };

    resizeCanvas();
    window.addEventListener('resize', resizeCanvas);

    if (initWebGL()) {
      animate();
    }

    return () => {
      window.removeEventListener('resize', resizeCanvas);
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [rotation]);

  // Mouse controls for camera rotation only
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (e.buttons === 1) { // Left mouse button
        setRotation(prev => prev + e.movementX * 0.01);
      }
    };

    window.addEventListener('mousemove', handleMouseMove);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
    };
  }, []);

  return (
    <div
      className="spiral-galaxy-animation"
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        width: '100vw',
        height: '100vh',
        zIndex,
        pointerEvents: 'none',
        overflow: 'hidden',
      }}
    >
      <canvas
        ref={canvasRef}
        style={{
          width: '100%',
          height: '100%',
          pointerEvents: 'auto',
        }}
      />
    </div>
  );
};

export default SpiralGalaxyAnimation; 
