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
  isNebula?: boolean;
  orbitRadius?: number;
  orbitAngle?: number;
  orbitSpeed?: number;
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
    
    // Determine particle type based on brightness and color
    bool isNebula = v_brightness > 0.15 && (v_color.r > 0.7 && v_color.g > 0.7); // Detect yellow nebula
    
    float alpha;
    if (isNebula) {
      // Very soft, diffuse nebula effect for better blending
      alpha = exp(-dist * dist * 2.0) * v_brightness;
      
      // Add extended outer glow for smoother edges
      float outerGlow = exp(-dist * dist * 0.8) * v_brightness * 0.6;
      alpha += outerGlow;
      
      // Add very soft halo for seamless blending
      float halo = exp(-dist * dist * 0.3) * v_brightness * 0.3;
      alpha += halo;
    } else if (v_brightness < 0.3) {
      // Standard dust particles
      alpha = exp(-dist * dist * 2.0) * v_brightness;
      
      // Add soft glow for dust effect
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
    armPitch: Math.PI / 6, // Much tighter spiral arms - increased from π/18 to π/6
    inclination: 80 * Math.PI / 180, // 77° from face-on
    orientation: 20 * Math.PI / 180, // Position angle
    particleCount: 2000, // Adjusted for extended distribution
    rotationSpeed: 0.00001,
    timeScale: 10000000 // 1 sim second = 10 million years
  };

  // Generate spiral galaxy structure with natural distribution and realistic colors
  const generateGalaxyParticles = (): StarParticle[] => {
    const particles: StarParticle[] = [];
    
    // Increase particle count significantly for more detail
    const totalParticles = galaxyParams.particleCount * 2;
    
        // Generate central gas cloud - bright yellow
    const centralCloudParticles = 500;
    const cloudRadius = galaxyParams.coreRadius * 2.5; // Larger radius
    
    for (let i = 0; i < centralCloudParticles; i++) {
      // Create spherical distribution with softer edges
      const u = Math.random();
      const r = cloudRadius * Math.pow(u, 0.6); // Power distribution for softer edges
      const theta = Math.random() * 2 * Math.PI;
      const phi = Math.acos(2 * Math.random() - 1); // Uniform distribution on sphere
      
      const x = r * Math.sin(phi) * Math.cos(theta);
      const y = r * Math.sin(phi) * Math.sin(theta);
      const z = r * Math.cos(phi) * 0.3; // Flatten slightly for disk-like appearance
      // Distance-based color and brightness for softer edges, denser at the core
      // Use a steeper falloff for higher core density
      const distanceFromCenter = r / cloudRadius;
      const edgeFalloff = 1.0 - Math.pow(distanceFromCenter, 3); // Cubic falloff for denser core
      // Bright yellow gas cloud colors with edge variation
      const yellowVariation = Math.random() * 50;
      const edgeColorShift = distanceFromCenter * 30; // Shift toward orange at edges
      const nebulaColor = `rgb(${Math.floor(255 - yellowVariation - edgeColorShift)}, ${Math.floor(255 - yellowVariation * 0.3 - edgeColorShift * 0.5)}, ${Math.floor(100 + yellowVariation - edgeColorShift * 0.3)})`;
      
      // Brightness fades toward edges
      const baseBrightness = 0.2 + Math.random() * 0.3;
      const brightness = baseBrightness * (0.3 + edgeFalloff * 0.7);
      
      // Size varies with distance for smoother blending
      const baseSize = 1.2 + Math.random() * 2.5;
      const size = baseSize * (0.8 + edgeFalloff * 0.4);
      
      particles.push({
        x,
        y,
        z,
        brightness,
        color: nebulaColor,
        size,
        temperature: 6000,
        isNebula: true
      });
    }
    
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

    // Generate nebula/dust particles for smoke effect - distributed across multiple spiral features
    const nebulaParticles = Math.floor(totalParticles * 0.7);
    for (let i = 0; i < nebulaParticles; i++) {
      const u = Math.random();
      const v = Math.random();
      
      // Focus nebula in disk regions with exponential falloff
      let r: number;
      if (u < 0.85) {
        r = galaxyParams.coreRadius + (galaxyParams.diskRadius - galaxyParams.coreRadius) * Math.sqrt(v);
      } else {
        r = galaxyParams.diskRadius + (galaxyParams.diskRadius * 0.4) * Math.pow(v, 3);
      }
      
      const baseTheta = Math.random() * 2 * Math.PI;
      
      // Multiple spiral arm systems like real Andromeda
      let maxArmStrength = 0;
      
             // First pair of spiral arms (2 arms)
       const arm1SpiralFactor = 2;
       const arm1SpiralAngle = arm1SpiralFactor * (baseTheta + galaxyParams.armPitch * 2.0 * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
       const arm1ArmAngle = arm1SpiralAngle % (2 * Math.PI / arm1SpiralFactor);
       const arm1DistanceToArm = Math.min(arm1ArmAngle, (2 * Math.PI / arm1SpiralFactor) - arm1ArmAngle);
       const arm1ArmWidth = Math.PI / 4;
       const arm1ArmStrength = Math.exp(-Math.pow(arm1DistanceToArm / arm1ArmWidth, 2)) * 1.0;
       maxArmStrength = Math.max(maxArmStrength, arm1ArmStrength);
       
       // Second pair of spiral arms (2 arms, offset)
       const arm2SpiralFactor = 2;
       const arm2SpiralAngle = arm2SpiralFactor * (baseTheta + Math.PI/3 + galaxyParams.armPitch * 2.3 * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
       const arm2ArmAngle = arm2SpiralAngle % (2 * Math.PI / arm2SpiralFactor);
       const arm2DistanceToArm = Math.min(arm2ArmAngle, (2 * Math.PI / arm2SpiralFactor) - arm2ArmAngle);
       const arm2ArmWidth = Math.PI / 4;
       const arm2ArmStrength = Math.exp(-Math.pow(arm2DistanceToArm / arm2ArmWidth, 2)) * 0.8;
       maxArmStrength = Math.max(maxArmStrength, arm2ArmStrength);
       
       // Third pair of spiral arms (2 arms, different offset)
       const arm3SpiralFactor = 2;
       const arm3SpiralAngle = arm3SpiralFactor * (baseTheta + 2*Math.PI/3 + galaxyParams.armPitch * 2.6 * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
       const arm3ArmAngle = arm3SpiralAngle % (2 * Math.PI / arm3SpiralFactor);
       const arm3DistanceToArm = Math.min(arm3ArmAngle, (2 * Math.PI / arm3SpiralFactor) - arm3ArmAngle);
       const arm3ArmWidth = Math.PI / 4;
       const arm3ArmStrength = Math.exp(-Math.pow(arm3DistanceToArm / arm3ArmWidth, 2)) * 0.7;
       maxArmStrength = Math.max(maxArmStrength, arm3ArmStrength);
      
      // Ring-like structures (common in real galaxies)
      const ringRadius1 = galaxyParams.diskRadius * 0.3;
      const ringRadius2 = galaxyParams.diskRadius * 0.6;
      const ringRadius3 = galaxyParams.diskRadius * 0.85;
      
      const ringStrength1 = Math.exp(-Math.pow((r - ringRadius1) / 20, 2)) * 0.4;
      const ringStrength2 = Math.exp(-Math.pow((r - ringRadius2) / 25, 2)) * 0.3;
      const ringStrength3 = Math.exp(-Math.pow((r - ringRadius3) / 30, 2)) * 0.25;
      
      maxArmStrength = Math.max(maxArmStrength, ringStrength1, ringStrength2, ringStrength3);
      
             // Random spiral spurs and fragments - with more curvature, clockwise
       for (let spurIndex = 0; spurIndex < 6; spurIndex++) {
         const spurOffset = (spurIndex / 6) * 2 * Math.PI + Math.random() * Math.PI/3;
         const spurPitch = galaxyParams.armPitch * (1.5 + Math.random() * 2.0); // Much more curved spurs
         const spurFactor = 1 + Math.random() * 2;
         const spurSpiralAngle = spurFactor * (baseTheta + spurOffset + spurPitch * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
         const spurArmAngle = spurSpiralAngle % (2 * Math.PI / spurFactor);
         const spurDistanceToArm = Math.min(spurArmAngle, (2 * Math.PI / spurFactor) - spurArmAngle);
         const spurArmWidth = Math.PI / (10 + Math.random() * 5);
         const spurStrength = Math.exp(-Math.pow(spurDistanceToArm / spurArmWidth, 2)) * (0.2 + Math.random() * 0.3);
         maxArmStrength = Math.max(maxArmStrength, spurStrength);
       }
      
      // Spiral structure fades with radius
      const spiralFalloff = Math.exp(-r / (galaxyParams.diskRadius * 1.3));
      maxArmStrength *= spiralFalloff;
      
      // Nebula threshold - more permissive to fill spiral structures
      const nebulaThreshold = 0.75 + maxArmStrength * 0.25;
      if (Math.random() > nebulaThreshold) continue;
      
      const perturbation = (Math.random() - 0.5) * 20;
      const theta = baseTheta + perturbation / Math.max(r, 15);
      
      const x = r * Math.cos(theta);
      const y = r * Math.sin(theta);
      
      const nebulaHeight = 18 * Math.exp(-r / (galaxyParams.diskRadius * 0.8));
      const z = (Math.random() - 0.5) * Math.max(nebulaHeight, 2);
      
      // Nebula colors based on spiral strength
      let nebulaColor: string;
      let temperature: number;
      
      if (maxArmStrength > 0.5 && Math.random() < 0.6) {
        // Bright HII regions in strong spiral features
        nebulaColor = `rgb(${255}, ${Math.floor(70 + Math.random() * 130)}, ${Math.floor(90 + Math.random() * 110)})`;
        temperature = 12000;
      } else if (maxArmStrength > 0.3 && Math.random() < 0.4) {
        // Blue reflection nebulae
        nebulaColor = `rgb(${Math.floor(70 + Math.random() * 130)}, ${Math.floor(120 + Math.random() * 110)}, ${255})`;
        temperature = 8000;
      } else {
        // Dust lanes and dark nebulae
        const dustRed = Math.floor(60 + Math.random() * 90);
        const dustGreen = Math.floor(25 + Math.random() * 55);
        const dustBlue = Math.floor(10 + Math.random() * 40);
        nebulaColor = `rgb(${dustRed}, ${dustGreen}, ${dustBlue})`;
        temperature = 150;
      }
      
      const brightness = 0.05 + Math.random() * 0.15;
      const size = 1.2 + Math.random() * 3.8;
      
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
      // Generate radius with realistic distribution
      const u = Math.random();
      const v = Math.random();
      
      const maxRadius = galaxyParams.diskRadius * 1.8;
      let r: number;
      
      if (u < 0.8) {
        // Most particles in main disk
        r = galaxyParams.coreRadius + (galaxyParams.diskRadius - galaxyParams.coreRadius) * Math.pow(v, 0.8);
      } else {
        // Extended halo with steep falloff
        const haloFactor = Math.pow(v, 5);
        r = galaxyParams.diskRadius + (maxRadius - galaxyParams.diskRadius) * haloFactor;
      }
      
      const baseTheta = Math.random() * 2 * Math.PI;
      
      // Calculate spiral arm strength using multiple arm systems
      let totalSpiralStrength = 0;
      
             // First pair of spiral arms (2 arms)
       const arm1SpiralFactor = 2;
       const arm1SpiralAngle = arm1SpiralFactor * (baseTheta + galaxyParams.armPitch * 2.0 * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
       const arm1NormalizedAngle = arm1SpiralAngle % (2 * Math.PI / arm1SpiralFactor);
       const arm1DistanceToArm = Math.min(arm1NormalizedAngle, (2 * Math.PI / arm1SpiralFactor) - arm1NormalizedAngle);
       const arm1ArmWidth = Math.PI / 8;
       const arm1ArmStrength = Math.exp(-Math.pow(arm1DistanceToArm / arm1ArmWidth, 2)) * 1.0;
       totalSpiralStrength += arm1ArmStrength;
       
       // Second pair of spiral arms (2 arms, offset)
       const arm2SpiralFactor = 2;
       const arm2SpiralAngle = arm2SpiralFactor * (baseTheta + Math.PI/3 + galaxyParams.armPitch * 2.3 * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
       const arm2NormalizedAngle = arm2SpiralAngle % (2 * Math.PI / arm2SpiralFactor);
       const arm2DistanceToArm = Math.min(arm2NormalizedAngle, (2 * Math.PI / arm2SpiralFactor) - arm2NormalizedAngle);
       const arm2ArmWidth = Math.PI / 8;
       const arm2ArmStrength = Math.exp(-Math.pow(arm2DistanceToArm / arm2ArmWidth, 2)) * 0.9;
       totalSpiralStrength += arm2ArmStrength;
       
       // Third pair of spiral arms (2 arms, different offset)
       const arm3SpiralFactor = 2;
       const arm3SpiralAngle = arm3SpiralFactor * (baseTheta + 2*Math.PI/3 + galaxyParams.armPitch * 2.6 * Math.log(Math.max(r, galaxyParams.coreRadius) / galaxyParams.coreRadius));
       const arm3NormalizedAngle = arm3SpiralAngle % (2 * Math.PI / arm3SpiralFactor);
       const arm3DistanceToArm = Math.min(arm3NormalizedAngle, (2 * Math.PI / arm3SpiralFactor) - arm3NormalizedAngle);
       const arm3ArmWidth = Math.PI / 8;
       const arm3ArmStrength = Math.exp(-Math.pow(arm3DistanceToArm / arm3ArmWidth, 2)) * 0.8;
       totalSpiralStrength += arm3ArmStrength;
      
      // Multiple ring structures
      const ringRadius1 = galaxyParams.diskRadius * 0.25;
      const ringRadius2 = galaxyParams.diskRadius * 0.5;
      const ringRadius3 = galaxyParams.diskRadius * 0.75;
      const ringRadius4 = galaxyParams.diskRadius * 0.95;
      
      const ringStrength1 = Math.exp(-Math.pow((r - ringRadius1) / 15, 2)) * 0.5;
      const ringStrength2 = Math.exp(-Math.pow((r - ringRadius2) / 20, 2)) * 0.4;
      const ringStrength3 = Math.exp(-Math.pow((r - ringRadius3) / 25, 2)) * 0.35;
      const ringStrength4 = Math.exp(-Math.pow((r - ringRadius4) / 30, 2)) * 0.3;
      
      totalSpiralStrength += ringStrength1 + ringStrength2 + ringStrength3 + ringStrength4;
      
             // Random spiral spurs and substructure (like real Andromeda) - with more curvature, clockwise
       for (let spurIndex = 0; spurIndex < 8; spurIndex++) {
         const spurOffset = (spurIndex / 8) * 2 * Math.PI + (Math.random() - 0.5) * Math.PI/2;
         const spurPitch = galaxyParams.armPitch * (1.5 + Math.random() * 2.5); // Much more curved spurs
         const spurFactor = 1 + Math.random() * 3;
         const spurRadius = Math.max(r, galaxyParams.coreRadius);
         const spurSpiralAngle = spurFactor * (baseTheta + spurOffset + spurPitch * Math.log(spurRadius / galaxyParams.coreRadius));
         const spurNormalizedAngle = spurSpiralAngle % (2 * Math.PI / spurFactor);
         const spurDistanceToArm = Math.min(spurNormalizedAngle, (2 * Math.PI / spurFactor) - spurNormalizedAngle);
         const spurArmWidth = Math.PI / (12 + Math.random() * 8);
         const spurStrength = Math.exp(-Math.pow(spurDistanceToArm / spurArmWidth, 2)) * (0.15 + Math.random() * 0.4);
         totalSpiralStrength += spurStrength;
       }
      
      // Spiral structure fades with radius
      const spiralFalloff = Math.exp(-r / (galaxyParams.diskRadius * 1.4));
      totalSpiralStrength *= spiralFalloff;
      
      // Density thresholding with much lower base density
      let densityThreshold: number;
      if (r <= galaxyParams.coreRadius) {
        // Core region
        const baseDensity = 0.45;
        const spiralBoost = totalSpiralStrength * 0.4;
        densityThreshold = baseDensity + spiralBoost;
      } else if (r <= galaxyParams.diskRadius) {
        // Main disk - very low base, high spiral enhancement
        const radialFalloff = Math.exp(-Math.pow((r - galaxyParams.coreRadius) / (galaxyParams.diskRadius - galaxyParams.coreRadius), 1.2));
        const baseDensity = 0.04 * radialFalloff; // Even lower base density
        const spiralBoost = totalSpiralStrength * 1.5 * radialFalloff; // Stronger spiral enhancement
        densityThreshold = baseDensity + spiralBoost;
      } else {
        // Extended regions - extremely sparse
        const extendedDistance = r - galaxyParams.diskRadius;
        const extendedFactor = Math.exp(-Math.pow(extendedDistance / (galaxyParams.diskRadius * 0.15), 2));
        const baseDensity = 0.01 * extendedFactor;
        const spiralBoost = totalSpiralStrength * 0.05 * extendedFactor;
        densityThreshold = baseDensity + spiralBoost;
      }
      
      // Apply density threshold
      if (Math.random() > densityThreshold) continue;
      
      // Minimal perturbation for cleaner spiral structure
      const perturbationScale = Math.min(12, r * 0.025);
      const randomPerturbation = (Math.random() - 0.5) * perturbationScale;
      
      // Strong bias toward spiral features
      const spiralBias = totalSpiralStrength > 0.1 ? (Math.random() - 0.5) * perturbationScale * 0.6 * totalSpiralStrength : 0;
      const totalPerturbation = randomPerturbation + spiralBias;
      
      const theta = baseTheta + totalPerturbation / Math.max(r, 20);
      
      const x = r * Math.cos(theta);
      const y = r * Math.sin(theta);
      
      // Vertical distribution
      const baseScaleHeight = 10;
      const scaleHeight = baseScaleHeight * Math.exp(-r / (galaxyParams.diskRadius * 0.4));
      const z = (Math.random() - 0.5) * Math.max(scaleHeight, 1);
      
      // Stellar populations based on spiral strength
      let temperature: number;
      let brightness: number;
      
      if (r <= galaxyParams.coreRadius * 2) {
        // Core region
        if (Math.random() < 0.8) {
          temperature = 3000 + Math.random() * 2000;
          brightness = 0.6 + Math.random() * 0.4;
        } else {
          temperature = 15000 + Math.random() * 15000;
          brightness = 0.8 + Math.random() * 0.2;
        }
      } else if (r <= galaxyParams.diskRadius && totalSpiralStrength > 0.3) {
        // Strong spiral features - young hot stars
        if (Math.random() < 0.8) {
          temperature = 8000 + Math.random() * 15000;
          brightness = 0.6 + Math.random() * 0.4;
        } else {
          temperature = 4000 + Math.random() * 4000;
          brightness = 0.4 + Math.random() * 0.4;
        }
      } else if (r <= galaxyParams.diskRadius && totalSpiralStrength > 0.1) {
        // Moderate spiral features
        temperature = 4000 + Math.random() * 6000;
        brightness = 0.3 + Math.random() * 0.4;
      } else if (r <= galaxyParams.diskRadius) {
        // Inter-arm regions - very dim
        temperature = 3500 + Math.random() * 2500;
        brightness = 0.15 + Math.random() * 0.25;
      } else {
        // Halo - extremely dim
        temperature = 3000 + Math.random() * 1500;
        const haloDistance = r - galaxyParams.diskRadius;
        brightness = Math.exp(-haloDistance / (galaxyParams.diskRadius * 0.1)) * (0.01 + Math.random() * 0.05);
      }
      
      // Apply brightness falloff
      if (r <= galaxyParams.diskRadius) {
        brightness *= Math.exp(-Math.pow(r / (galaxyParams.diskRadius * 0.7), 1.8));
      }
      
      // Size based on brightness and temperature
      const baseSize = 0.15 + Math.random() * 0.9;
      const tempFactor = Math.max(0.3, Math.min(2.2, temperature / 6000));
      const size = baseSize * (0.2 + brightness * 1.8) * tempFactor;
      
      const [r_color, g_color, b_color] = temperatureToColor(temperature);
      const color = `rgb(${Math.round(r_color * 255)}, ${Math.round(g_color * 255)}, ${Math.round(b_color * 255)})`;
      
      particles.push({
        x,
        y,
        z,
        brightness: Math.min(brightness, 1),
        color,
        size: Math.max(size, 0.05),
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
    let rotatedX: number, rotatedY: number, rotatedZ: number;
    
    // Apply standard galaxy rotation (differential rotation)
    const r = Math.sqrt(particle.x * particle.x + particle.y * particle.y);
    let rotationRate = galaxyParams.rotationSpeed;
    
    if (r > galaxyParams.coreRadius) {
      // Flat rotation curve in outer regions
      rotationRate = galaxyParams.rotationSpeed * galaxyParams.coreRadius / r;
    }
    
    const currentAngle = Math.atan2(particle.y, particle.x);
    const newAngle = currentAngle + rotationRate * time * timeSpeed;
    
    rotatedX = r * Math.cos(newAngle);
    rotatedY = r * Math.sin(newAngle);
    rotatedZ = particle.z;
    
    // Add subtle turbulent motion for nebula particles
    if (particle.isNebula) {
      const turbulenceScale = 0.5;
      const turbulenceX = Math.sin(time * 0.00005 + particle.x * 0.01) * turbulenceScale;
      const turbulenceY = Math.cos(time * 0.00005 + particle.y * 0.01) * turbulenceScale;
      
      rotatedX += turbulenceX;
      rotatedY += turbulenceY;
    }

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
      
      let size = particle.size * projected.scale;
      
      // Make nebula particles larger and more diffuse
      if (particle.isNebula) {
        size *= 1.5 + Math.sin(timeRef.current * 0.0001 + particle.x * 0.01) * 0.3;
      }
      
      sizes.push(Math.max(1.0, size));
      
      // Parse color
      const colorMatch = particle.color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
      if (colorMatch) {
        let r = parseInt(colorMatch[1]) / 255;
        let g = parseInt(colorMatch[2]) / 255;
        let b = parseInt(colorMatch[3]) / 255;
        
        // Add color variation for nebula particles
        if (particle.isNebula) {
          const colorShift = Math.sin(timeRef.current * 0.00005 + particle.y * 0.01) * 0.1;
          r = Math.max(0, Math.min(1, r + colorShift));
          g = Math.max(0, Math.min(1, g + colorShift * 0.5));
          b = Math.max(0, Math.min(1, b + colorShift * 0.8));
        }
        
        colors.push(r, g, b);
      } else {
        colors.push(1, 1, 1); // Default white
      }
      
      let alpha = particle.brightness * Math.min(1, projected.scale);
      
      // Add pulsing effect for nebula
      if (particle.isNebula) {
        const pulse = 0.8 + 0.2 * Math.sin(timeRef.current * 0.00008 + particle.x * 0.005 + particle.y * 0.003);
        alpha *= pulse;
      }
      
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
