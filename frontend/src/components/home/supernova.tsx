import React, { useEffect, useRef, useState } from 'react';

interface StarParticle {
  x: number;
  y: number;
  z: number;
  brightness: number;
  color: string;
  size: number;
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

export const SpiralGalaxyAnimation: React.FC<{ zIndex?: number }> = ({ zIndex = -1 }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animationRef = useRef<number>();
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
    inclination: 77 * Math.PI / 180, // 77° from face-on
    orientation: 35 * Math.PI / 180, // Position angle
    particleCount: 6000, // Adjusted for extended distribution
    rotationSpeed: 0.00001,
    timeScale: 10000000 // 1 sim second = 10 million years
  };

  // Generate spiral galaxy structure with natural distribution
  const generateGalaxyParticles = (): StarParticle[] => {
    const particles: StarParticle[] = [];
    
    // Increase particle count to account for extended distribution
    const totalParticles = galaxyParams.particleCount * 1.5;
    
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
       let densityThreshold: number;
       if (r <= galaxyParams.diskRadius) {
         // Main disk - strong spiral structure
         const baseDensity = 0.15; // Lower base to make arms more prominent
         const spiralBoost = combinedSpiral * 0.7; // Strong spiral enhancement
         densityThreshold = baseDensity + spiralBoost;
       } else {
         // Extended regions - much sparser, weaker spiral structure
         const extendedFactor = Math.exp(-(r - galaxyParams.diskRadius) / (galaxyParams.diskRadius * 0.4));
         const baseDensity = 0.05 * extendedFactor;
         const spiralBoost = combinedSpiral * 0.3 * extendedFactor;
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
      
      // Brightness calculation with more realistic falloff
      let brightness: number;
      if (r <= galaxyParams.diskRadius) {
        // Main disk brightness
        brightness = Math.exp(-r / (galaxyParams.diskRadius * 0.5)) * (0.2 + Math.random() * 0.8);
      } else {
        // Extended halo - much dimmer
        const haloDistance = r - galaxyParams.diskRadius;
        brightness = Math.exp(-haloDistance / (galaxyParams.diskRadius * 0.3)) * (0.05 + Math.random() * 0.15);
      }
      
      // Size variation based on brightness and distance
      const baseSize = 0.3 + Math.random() * 1.2;
      const size = baseSize * (0.5 + brightness * 1.5);
      
      particles.push({
        x,
        y,
        z,
        brightness: Math.min(brightness, 1),
        color: 'white',
        size: Math.max(size, 0.2),
      });
    }

    return particles;
  };

  // Color interpolation helper
  const interpolateColor = (color1: string, color2: string, factor: number): string => {
    const hex1 = color1.replace('#', '');
    const hex2 = color2.replace('#', '');
    
    const r1 = parseInt(hex1.substr(0, 2), 16);
    const g1 = parseInt(hex1.substr(2, 2), 16);
    const b1 = parseInt(hex1.substr(4, 2), 16);
    
    const r2 = parseInt(hex2.substr(0, 2), 16);
    const g2 = parseInt(hex2.substr(2, 2), 16);
    const b2 = parseInt(hex2.substr(4, 2), 16);
    
    const r = Math.round(r1 + (r2 - r1) * factor);
    const g = Math.round(g1 + (g2 - g1) * factor);
    const b = Math.round(b1 + (b2 - b1) * factor);
    
    return `rgb(${r}, ${g}, ${b})`;
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
      visible: screenX >= 0 && screenX <= canvasWidth && screenY >= 0 && screenY <= canvasHeight
    };
  };

  const [particles] = useState(() => generateGalaxyParticles());

  const animate = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const { width, height } = canvas;
    
    // Clear canvas with dark space background
    ctx.fillStyle = 'rgba(5, 5, 15, 1)';
    ctx.fillRect(0, 0, width, height);

    if (isPlaying) {
      timeRef.current += 16; // ~60fps
    }

    // Sort particles by z-depth for proper rendering
    const projectedParticles = particles
      .map(particle => ({
        particle,
        projected: projectParticle(particle, timeRef.current, width, height)
      }))
      .filter(p => p.projected.visible)
      .sort((a, b) => b.projected.z - a.projected.z);

    // Render particles as simple white dots
    projectedParticles.forEach(({ particle, projected }) => {
      const { x, y, scale } = projected;
      const size = particle.size * scale;
      const alpha = particle.brightness * Math.min(1, scale);

      ctx.save();
      ctx.globalAlpha = alpha;
      ctx.fillStyle = 'white';
      
      // Simple circular particle
      ctx.beginPath();
      ctx.arc(x, y, Math.max(0.5, size), 0, 2 * Math.PI);
      ctx.fill();
      
      ctx.restore();
    });

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

    animate();

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
