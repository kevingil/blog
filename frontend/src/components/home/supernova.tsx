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
  isSpiralArm?: boolean; // Flag for spiral arm particles
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
}

// WebGL shader sources
const vertexShaderSource = `
  attribute vec2 a_position;
  attribute float a_size;
  attribute vec3 a_color;
  attribute float a_brightness;
  attribute float a_spiralArm; // New attribute for spiral arm detection
  
  uniform mat3 u_transform;
  uniform vec2 u_resolution;
  
  varying vec3 v_color;
  varying float v_brightness;
  varying float v_spiralArm;
  
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
    v_spiralArm = a_spiralArm;
  }
`;

  const fragmentShaderSource = `
  precision mediump float;
  
  varying vec3 v_color;
  varying float v_brightness;
  varying float v_spiralArm;
  
  void main() {
    vec2 center = gl_PointCoord - 0.5;
    float dist = length(center);
    
    if (dist > 0.5) {
      discard;
    }
    
    // Determine particle type based on brightness and color
    bool isNebula = v_brightness > 0.15 && (v_color.r > 0.7 && v_color.g > 0.7); // Detect yellow nebula
    bool isSpiralArm = v_spiralArm > 0.5; // Detect spiral arm particles
    bool isCore = v_color.r > 0.95 && v_color.g > 0.8 && v_color.b < 0.7; // Detect core (deep yellow/orange)
    bool isBrightBlue = v_brightness > 0.6 && v_color.b > 0.6; // Bright blue stars
    bool isColorfulDust = v_brightness > 0.05 && v_brightness < 0.25 && (v_color.r > 0.4 || v_color.g > 0.3); // Colorful inter-arm regions
    
    float alpha;
    vec3 outColor = v_color;
    if (isNebula) {
      // Very soft, diffuse nebula effect for better blending
      alpha = exp(-dist * dist * 2.0) * v_brightness;
      float outerGlow = exp(-dist * dist * 0.8) * v_brightness * 0.6;
      alpha += outerGlow;
      float halo = exp(-dist * dist * 0.3) * v_brightness * 0.3;
      alpha += halo;
    } else if (isSpiralArm || isCore) {
      // Enhanced rendering for spiral arm and core stars: warm glow
      alpha = exp(-dist * dist * 6.0) * v_brightness * 1.3;
      float spiralGlow = exp(-dist * dist * 1.5) * v_brightness * 0.9;
      alpha += spiralGlow;
      float halo = exp(-dist * dist * 0.4) * v_brightness * 0.6;
      alpha += halo;
      // Add warm tint to the glow/halo
      float warmth = isCore ? 0.4 : 0.25;
      outColor = mix(v_color, vec3(1.0, 0.85, 0.3), warmth * (1.0 - dist));
    } else if (isColorfulDust) {
      alpha = exp(-dist * dist * 1.5) * v_brightness * 1.5;
      float dustGlow = exp(-dist * dist * 0.6) * v_brightness * 1.2;
      alpha += dustGlow;
      float softHalo = exp(-dist * dist * 0.25) * v_brightness * 0.8;
      alpha += softHalo;
    } else if (v_brightness < 0.3) {
      alpha = exp(-dist * dist * 4.0) * v_brightness * 0.6;
      float softGlow = exp(-dist * dist * 1.0) * v_brightness * 0.3;
      alpha += softGlow;
    } else {
      alpha = exp(-dist * dist * 5.0) * v_brightness;
      float glow = exp(-dist * dist * 2.0) * v_brightness * 0.4;
      alpha += glow;
    }
    gl_FragColor = vec4(outColor, alpha);
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
    spiralArm: WebGLBuffer | null;
  }>({
    position: null,
    size: null,
    color: null,
    brightness: null,
    spiralArm: null
  });
  const [isPlaying] = useState(true);
  const timeSpeed = 3; // Fixed 3x speed
  // Responsive zoom and offsets
  const [zoom, setZoom] = useState(1.2);
  const zoomRef = useRef(1.2);
  const [centerOffset, setCenterOffset] = useState({ x: 175, y: -100 });
  const centerOffsetRef = useRef({ x: 175, y: -100 });
  const [rotation, setRotation] = useState(0);
  const timeRef = useRef(0);
  const lastFrameTimeRef = useRef(0);

  // Galaxy parameters based on Andromeda M31
  const galaxyParams: GalaxyParams = {
    coreRadius: 50, // Scaled for viewport
    diskRadius: 350, // Slightly smaller main disk for better spiral definition
    armPitch: Math.PI / 6, // Much tighter spiral arms - increased from π/18 to π/6
    inclination: 80 * Math.PI / 180, // 77° from face-on
    orientation: 10 * Math.PI / 180, // Position angle
    particleCount: 1000, // Adjusted for extended distribution
    rotationSpeed: 0.0001,
  };

  // Responsive effect for zoom and centering
  useEffect(() => {
    function handleResize() {
      const width = window.innerWidth;
      if (width <= 640) { // sm
        setZoom(0.6);
        zoomRef.current = 0.6;
        setCenterOffset({ x: 40, y: -50 });
        centerOffsetRef.current = { x: 40, y: -50 };
      } else if (width <= 1024) { // md
        setZoom(0.7);
        zoomRef.current = 0.8;
        setCenterOffset({ x: 40, y: -60 });
        centerOffsetRef.current = { x: 100, y: -60 };
      } else {
        setZoom(1.0);
        zoomRef.current = 1.0;
        setCenterOffset({ x: 120, y: -70 });
        centerOffsetRef.current = { x: 150, y: -70 };
      }
    }
    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

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
      const yellowVariation = Math.random() * 20;
      const edgeColorShift = distanceFromCenter * 20;
      // Blend from bright yellow at edge to almost white at center
      const t = 1.0 - edgeFalloff; // 0 at center, 1 at edge
      const coreR = 255;
      const coreG = Math.floor(242 * t + 255 * (1 - t) - yellowVariation - edgeColorShift);
      const coreB = Math.floor(0 * t + 220 * (1 - t));
      const nebulaColor = `rgb(${coreR}, ${coreG}, ${coreB})`;
      
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
      
      // PROPER Andromeda-style logarithmic spiral arms with multiple turns
      let maxArmStrength = 0;
      
      // Main spiral arm parameters (like real Andromeda)
      const numMainArms = 2; // Andromeda has 2 prominent arms
      const armTightness = 0.25; // Controls how tightly wound the spiral is (smaller = tighter)
      
      for (let armIndex = 0; armIndex < numMainArms; armIndex++) {
        // Each arm starts at a different angle
        const armStartAngle = (armIndex * Math.PI); // 180° apart
        
        // Logarithmic spiral: r = a * exp(b * theta)
        // Rearranged: theta = (1/b) * ln(r/a) 
        const spiralConstant = galaxyParams.coreRadius * 0.8;
        
        // Calculate the theoretical angle for this radius on this spiral arm (reverse direction)
        const theoreticalTheta = -(1 / armTightness) * Math.log(Math.max(r, spiralConstant) / spiralConstant);
        const spiralAngle = theoreticalTheta + armStartAngle;
        
        // Find angular distance from particle to spiral arm
        let angleDiff = baseTheta - spiralAngle;
        
        // Normalize to [-π, π] and handle multiple turns
        angleDiff = ((angleDiff % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI) - Math.PI;
        
        // Convert to arm strength
        const armWidth = 0.4; // Arm width in radians
        const armStrength = Math.exp(-Math.pow(angleDiff / armWidth, 2)) * 5.0;
        maxArmStrength = Math.max(maxArmStrength, armStrength);
      }
      
      // Secondary inner spiral structure
      const numSecondaryArms = 2;
      const secondaryTightness = 0.18; // Tighter winding for inner structure
      
      for (let armIndex = 0; armIndex < numSecondaryArms; armIndex++) {
        const armStartAngle = (armIndex * Math.PI) + Math.PI/2; // 90° offset from main arms
        const spiralConstant = galaxyParams.coreRadius * 0.4;
        
        const theoreticalTheta = -(1 / secondaryTightness) * Math.log(Math.max(r, spiralConstant) / spiralConstant);
        const spiralAngle = theoreticalTheta + armStartAngle;
        
        let angleDiff = baseTheta - spiralAngle;
        angleDiff = ((angleDiff % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI) - Math.PI;
        
        const armWidth = 0.3;
        const armStrength = Math.exp(-Math.pow(angleDiff / armWidth, 2)) * 3.0;
        maxArmStrength = Math.max(maxArmStrength, armStrength);
      }
      
      // Outer spiral structure (like Andromeda's extended arms)
      const numOuterArms = 2;
      const outerTightness = 0.35; // Looser winding for outer regions
      
      if (r > galaxyParams.coreRadius * 1.5) {
        for (let armIndex = 0; armIndex < numOuterArms; armIndex++) {
          const armStartAngle = (armIndex * Math.PI) + Math.PI/4;
          const spiralConstant = galaxyParams.coreRadius * 1.2;
          
          const theoreticalTheta = -(1 / outerTightness) * Math.log(Math.max(r, spiralConstant) / spiralConstant);
          const spiralAngle = theoreticalTheta + armStartAngle;
          
          let angleDiff = baseTheta - spiralAngle;
          angleDiff = ((angleDiff % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI) - Math.PI;
          
          const armWidth = 0.5;
          const armStrength = Math.exp(-Math.pow(angleDiff / armWidth, 2)) * 2.0;
          maxArmStrength = Math.max(maxArmStrength, armStrength);
        }
      }
      
      // Add spiral spurs and fragments
      for (let spurIndex = 0; spurIndex < 12; spurIndex++) {
        const spurStartAngle = (spurIndex / 12) * 2 * Math.PI;
        const spurTightness = 0.15 + Math.random() * 0.25;
        const spurConstant = galaxyParams.coreRadius * (0.3 + Math.random() * 0.6);
        
        const theoreticalTheta = -(1 / spurTightness) * Math.log(Math.max(r, spurConstant) / spurConstant);
        const spiralAngle = theoreticalTheta + spurStartAngle;
        
        let angleDiff = baseTheta - spiralAngle;
        angleDiff = ((angleDiff % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI) - Math.PI;
        
        const spurWidth = 0.2 + Math.random() * 0.15;
        const spurStrength = Math.exp(-Math.pow(angleDiff / spurWidth, 2)) * (1.0 + Math.random() * 1.0);
        maxArmStrength = Math.max(maxArmStrength, spurStrength);
      }
      
      // Spiral structure fades with radius
      const spiralFalloff = Math.exp(-Math.pow(r / (galaxyParams.diskRadius * 1.1), 1.8));
      maxArmStrength *= spiralFalloff;
      
      // Nebula threshold - more permissive for inter-arm colorful regions
      let nebulaThreshold: number;
      if (maxArmStrength < 0.2) {
        // More permissive for inter-arm colorful dust/nebula
        nebulaThreshold = 0.65 + maxArmStrength * 0.2;
      } else {
        // Standard threshold for spiral arm nebula
        nebulaThreshold = 0.75 + maxArmStrength * 0.25;
      }
      if (Math.random() > nebulaThreshold) continue;
      
      const perturbation = (Math.random() - 0.5) * 20;
      const theta = baseTheta + perturbation / Math.max(r, 15);
      
      const x = r * Math.cos(theta);
      const y = r * Math.sin(theta);
      
      const nebulaHeight = 18 * Math.exp(-r / (galaxyParams.diskRadius * 0.8));
      const z = (Math.random() - 0.5) * Math.max(nebulaHeight, 2);
      
      // Nebula colors based on spiral strength and location (Andromeda palette, warmer)
      let nebulaColor: string;
      let temperature: number;
      if (r <= galaxyParams.coreRadius * 1.5) {
        // Core: deep yellow/orange
        const g = 210 + Math.floor(Math.random() * 16);
        const b = 80 + Math.floor(Math.random() * 61);
        nebulaColor = `rgb(255,${g},${b})`;
        temperature = 6000;
      } else if (maxArmStrength > 1.0) {
        // Spiral arms: mostly warm yellow/orange, some yellow-white, rare blue-white/pink
        const rand = Math.random();
        if (rand < 0.7) {
          // Warm yellow/orange
          const g = 210 + Math.floor(Math.random() * 31);
          const b = 100 + Math.floor(Math.random() * 81);
          nebulaColor = `rgb(255,${g},${b})`;
          temperature = 7000;
        } else if (rand < 0.9) {
          // Yellow-white
          const g = 240 + Math.floor(Math.random() * 6);
          const b = 180 + Math.floor(Math.random() * 41);
          nebulaColor = `rgb(255,${g},${b})`;
          temperature = 9000;
        } else if (rand < 0.95) {
          // Blue-white (rare)
          const rVal = 180 + Math.floor(Math.random() * 41);
          const gVal = 210 + Math.floor(Math.random() * 31);
          nebulaColor = `rgb(${rVal},${gVal},255)`;
          temperature = 12000;
        } else {
          // Pink HII region (very rare)
          const g = 180 + Math.floor(Math.random() * 41);
          const b = 220 + Math.floor(Math.random() * 36);
          nebulaColor = `rgb(255,${g},${b})`;
          temperature = 10000;
        }
      } else if (r > galaxyParams.diskRadius * 0.9) {
        // Outer halo: faint, redder
        const rVal = 150 + Math.floor(Math.random() * 51);
        const gVal = 120 + Math.floor(Math.random() * 61);
        const bVal = 120 + Math.floor(Math.random() * 61);
        nebulaColor = `rgb(${rVal},${gVal},${bVal})`;
        temperature = 4000;
      } else if (maxArmStrength < 0.3 && r > galaxyParams.coreRadius * 1.5) {
        // Inter-arm/dust lanes: brown/tan
        const rVal = 120 + Math.floor(Math.random() * 61);
        const gVal = 90 + Math.floor(Math.random() * 51);
        const bVal = 60 + Math.floor(Math.random() * 41);
        nebulaColor = `rgb(${rVal},${gVal},${bVal})`;
        temperature = 1200;
      } else {
        // Standard dust lanes and dark nebulae
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
      
      // PROPER Andromeda-style logarithmic spiral arms (matching nebula generation)
      let totalSpiralStrength = 0;
      
      // Main spiral arms
      const numMainArms = 2;
      const armTightness = 0.25;
      
      for (let armIndex = 0; armIndex < numMainArms; armIndex++) {
        const armStartAngle = (armIndex * Math.PI);
        const spiralConstant = galaxyParams.coreRadius * 0.8;
        
        const theoreticalTheta = -(1 / armTightness) * Math.log(Math.max(r, spiralConstant) / spiralConstant);
        const spiralAngle = theoreticalTheta + armStartAngle;
        
        let angleDiff = baseTheta - spiralAngle;
        angleDiff = ((angleDiff % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI) - Math.PI;
        
        const armWidth = 0.3; // Slightly narrower for stars
        const armStrength = Math.exp(-Math.pow(angleDiff / armWidth, 2)) * 5.0;
        totalSpiralStrength += armStrength;
      }
      
      // Secondary inner spiral structure
      const numSecondaryArms = 2;
      const secondaryTightness = 0.18;
      
      for (let armIndex = 0; armIndex < numSecondaryArms; armIndex++) {
        const armStartAngle = (armIndex * Math.PI) + Math.PI/2;
        const spiralConstant = galaxyParams.coreRadius * 0.4;
        
        const theoreticalTheta = -(1 / secondaryTightness) * Math.log(Math.max(r, spiralConstant) / spiralConstant);
        const spiralAngle = theoreticalTheta + armStartAngle;
        
        let angleDiff = baseTheta - spiralAngle;
        angleDiff = ((angleDiff % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI) - Math.PI;
        
        const armWidth = 0.25;
        const armStrength = Math.exp(-Math.pow(angleDiff / armWidth, 2)) * 3.0;
        totalSpiralStrength += armStrength;
      }
      
      // Outer spiral structure
      const numOuterArms = 2;
      const outerTightness = 0.35;
      
      if (r > galaxyParams.coreRadius * 1.5) {
        for (let armIndex = 0; armIndex < numOuterArms; armIndex++) {
          const armStartAngle = (armIndex * Math.PI) + Math.PI/4;
          const spiralConstant = galaxyParams.coreRadius * 1.2;
          
          const theoreticalTheta = -(1 / outerTightness) * Math.log(Math.max(r, spiralConstant) / spiralConstant);
          const spiralAngle = theoreticalTheta + armStartAngle;
          
          let angleDiff = baseTheta - spiralAngle;
          angleDiff = ((angleDiff % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI) - Math.PI;
          
          const armWidth = 0.4;
          const armStrength = Math.exp(-Math.pow(angleDiff / armWidth, 2)) * 2.0;
          totalSpiralStrength += armStrength;
        }
      }
      
      // Add spiral spurs and fragments (matching nebula generation)
      for (let spurIndex = 0; spurIndex < 12; spurIndex++) {
        const spurStartAngle = (spurIndex / 12) * 2 * Math.PI;
        const spurTightness = 0.15 + Math.random() * 0.25;
        const spurConstant = galaxyParams.coreRadius * (0.3 + Math.random() * 0.6);
        
        const theoreticalTheta = -(1 / spurTightness) * Math.log(Math.max(r, spurConstant) / spurConstant);
        const spiralAngle = theoreticalTheta + spurStartAngle;
        
        let angleDiff = baseTheta - spiralAngle;
        angleDiff = ((angleDiff % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI) - Math.PI;
        
        const spurWidth = 0.15 + Math.random() * 0.1;
        const spurStrength = Math.exp(-Math.pow(angleDiff / spurWidth, 2)) * (1.0 + Math.random() * 1.0);
        totalSpiralStrength += spurStrength;
      }
      

      
      // Spiral structure fades with radius (matching nebula generation)
      const spiralFalloff = Math.exp(-Math.pow(r / (galaxyParams.diskRadius * 1.1), 1.8));
      totalSpiralStrength *= spiralFalloff;
      
      // Density thresholding with EXTREME spiral contrast
      let densityThreshold: number;
      if (r <= galaxyParams.coreRadius) {
        // Core region
        const baseDensity = 0.6;
        const spiralBoost = totalSpiralStrength * 0.3;
        densityThreshold = baseDensity + spiralBoost;
      } else if (r <= galaxyParams.diskRadius) {
        // Main disk - MASSIVE spiral enhancement, tiny base
        const radialFalloff = Math.exp(-Math.pow((r - galaxyParams.coreRadius) / (galaxyParams.diskRadius - galaxyParams.coreRadius), 1.2));
        const baseDensity = 0.005 * radialFalloff; // Extremely tiny base density
        const spiralBoost = totalSpiralStrength * 8.0 * radialFalloff; // MASSIVE spiral enhancement
        densityThreshold = baseDensity + spiralBoost;
      } else {
        // Extended regions - extremely sparse
        const extendedDistance = r - galaxyParams.diskRadius;
        const extendedFactor = Math.exp(-Math.pow(extendedDistance / (galaxyParams.diskRadius * 0.15), 2));
        const baseDensity = 0.002 * extendedFactor;
        const spiralBoost = totalSpiralStrength * 2.0 * extendedFactor;
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
      
      // Stellar populations with EXTREME spiral contrast
      let temperature: number;
      let brightness: number;
      
      if (r <= galaxyParams.coreRadius * 2) {
        // Core region
        if (Math.random() < 0.7) {
          temperature = 3000 + Math.random() * 2000;
          brightness = 0.7 + Math.random() * 0.3;
        } else {
          temperature = 15000 + Math.random() * 15000;
          brightness = 0.9 + Math.random() * 0.1;
        }
      } else if (r <= galaxyParams.diskRadius && totalSpiralStrength > 2.0) {
        // VERY strong spiral arms - brilliant blue supergiants
        temperature = 15000 + Math.random() * 20000; // Very hot blue stars
        brightness = 0.9 + Math.random() * 0.1; // Extremely bright
      } else if (r <= galaxyParams.diskRadius && totalSpiralStrength > 1.0) {
        // Strong spiral features - blue giants
        temperature = 10000 + Math.random() * 15000;
        brightness = 0.7 + Math.random() * 0.2;
      } else if (r <= galaxyParams.diskRadius && totalSpiralStrength > 0.3) {
        // Moderate spiral features - white stars
        temperature = 6000 + Math.random() * 8000;
        brightness = 0.5 + Math.random() * 0.3;
      } else if (r <= galaxyParams.diskRadius) {
        // Inter-arm regions - mix of dim stars and colorful dust/nebula
        if (Math.random() < 0.3) {
          // Reddish-brown dust lanes (like in Andromeda)
          temperature = 1500 + Math.random() * 1000; // Very cool dust
          brightness = 0.08 + Math.random() * 0.12;
        } else if (Math.random() < 0.5) {
          // Orange/amber older stellar populations
          temperature = 3500 + Math.random() * 1500; // K-type orange stars
          brightness = 0.04 + Math.random() * 0.08;
        } else {
          // Dim red dwarfs
          temperature = 2800 + Math.random() * 1200;
          brightness = 0.02 + Math.random() * 0.06;
        }
      } else {
        // Halo - virtually invisible
        temperature = 2500 + Math.random() * 1000;
        const haloDistance = r - galaxyParams.diskRadius;
        brightness = Math.exp(-haloDistance / (galaxyParams.diskRadius * 0.1)) * (0.001 + Math.random() * 0.01);
      }
      
      // Apply brightness falloff
      if (r <= galaxyParams.diskRadius) {
        brightness *= Math.exp(-Math.pow(r / (galaxyParams.diskRadius * 0.7), 1.8));
      }
      
      // Size based on brightness and temperature
      const baseSize = 0.15 + Math.random() * 0.9;
      const tempFactor = Math.max(0.3, Math.min(2.2, temperature / 6000));
      const size = baseSize * (0.2 + brightness * 1.8) * tempFactor;
      
      // Star color based on spiral strength and location (Andromeda palette, warmer)
      let color: string;
      if (r <= galaxyParams.coreRadius * 1.5) {
        // Core: deep yellow/orange
        const g = 210 + Math.floor(Math.random() * 16);
        const b = 80 + Math.floor(Math.random() * 61);
        color = `rgb(255,${g},${b})`;
      } else if (totalSpiralStrength > 1.0) {
        // Spiral arms: mostly warm yellow/orange, some yellow-white, rare blue-white/pink
        const rand = Math.random();
        if (rand < 0.7) {
          // Warm yellow/orange
          const g = 210 + Math.floor(Math.random() * 31);
          const b = 100 + Math.floor(Math.random() * 81);
          color = `rgb(255,${g},${b})`;
        } else if (rand < 0.9) {
          // Yellow-white
          const g = 240 + Math.floor(Math.random() * 6);
          const b = 180 + Math.floor(Math.random() * 41);
          color = `rgb(255,${g},${b})`;
        } else if (rand < 0.95) {
          // Blue-white (rare)
          const rVal = 180 + Math.floor(Math.random() * 41);
          const gVal = 210 + Math.floor(Math.random() * 31);
          color = `rgb(${rVal},${gVal},255)`;
        } else {
          // Pink HII region (very rare)
          const g = 180 + Math.floor(Math.random() * 41);
          const b = 220 + Math.floor(Math.random() * 36);
          color = `rgb(255,${g},${b})`;
        }
      } else if (r > galaxyParams.diskRadius * 0.9) {
        // Outer halo: faint, redder
        const rVal = 150 + Math.floor(Math.random() * 51);
        const gVal = 120 + Math.floor(Math.random() * 61);
        const bVal = 120 + Math.floor(Math.random() * 61);
        color = `rgb(${rVal},${gVal},${bVal})`;
      } else if (totalSpiralStrength < 0.3 && r > galaxyParams.coreRadius * 1.5) {
        // Inter-arm/dust lanes: brown/tan
        const rVal = 120 + Math.floor(Math.random() * 61);
        const gVal = 90 + Math.floor(Math.random() * 51);
        const bVal = 60 + Math.floor(Math.random() * 41);
        color = `rgb(${rVal},${gVal},${bVal})`;
      } else {
        // Standard faint star
        const [r_color, g_color, b_color] = temperatureToColor(temperature);
        color = `rgb(${Math.round(r_color * 255)}, ${Math.round(g_color * 255)}, ${Math.round(b_color * 255)})`;
      }
      
      // Determine if this is a spiral arm particle
      const isSpiralArm = totalSpiralStrength > 0.3;
      
      particles.push({
        x,
        y,
        z,
        brightness: Math.min(brightness, 1),
        color,
        size: Math.max(size, 0.05),
        temperature,
        isSpiralArm,
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
      brightness: gl.createBuffer(),
      spiralArm: gl.createBuffer()
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
    const scale = cameraDistance / (cameraDistance + inclinedZ) * zoomRef.current;
    
    // Responsive centering
    const screenX = canvasWidth / 2 - orientedX * scale + centerOffsetRef.current.x;
    const screenY = canvasHeight / 2 + orientedY * scale + centerOffsetRef.current.y;
    
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
      const currentTime = performance.now();
      if (lastFrameTimeRef.current === 0) {
        lastFrameTimeRef.current = currentTime;
      }
      const deltaTime = currentTime - lastFrameTimeRef.current;
      lastFrameTimeRef.current = currentTime;
      
      // Use smooth delta time instead of fixed increment
      timeRef.current += deltaTime * 0.16; // Match the original 16ms increment scaling
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
    const spiralArmFlags: number[] = [];

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
      
      // Add spiral arm flag (1.0 for spiral arm, 0.0 for non-spiral arm)
      spiralArmFlags.push(particle.isSpiralArm ? 1.0 : 0.0);
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

    // Spiral arm buffer
    gl.bindBuffer(gl.ARRAY_BUFFER, buffersRef.current.spiralArm);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(spiralArmFlags), gl.DYNAMIC_DRAW);
    const spiralArmLocation = gl.getAttribLocation(program, 'a_spiralArm');
    gl.enableVertexAttribArray(spiralArmLocation);
    gl.vertexAttribPointer(spiralArmLocation, 1, gl.FLOAT, false, 0, 0);

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
      className="spiral-galaxy-animation fixed top-0 left-0 w-full h-full pointer-events-none overflow-hidden z-[-1]"
      style={{ zIndex }}
    >
      <canvas
        ref={canvasRef}
        className="w-full h-full pointer-events-auto"
        style={{ width: '100%', height: '100%' }}
      />
    </div>
  );
};

export default SpiralGalaxyAnimation; 
