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
  const zoom = 0.85; // Fixed zoom level
  const [rotation, setRotation] = useState(0);
  const timeRef = useRef(0);

  // Galaxy parameters based on Andromeda M31
  const galaxyParams: GalaxyParams = {
    coreRadius: 50, // Scaled for viewport
    diskRadius: 400,
    armPitch: Math.PI / 20, // ~8-10 degrees
    inclination: 77 * Math.PI / 180, // 77° from face-on
    orientation: 35 * Math.PI / 180, // Position angle
    particleCount: 8000, // Reduced for better performance
    rotationSpeed: 0.0001,
    timeScale: 10000000 // 1 sim second = 10 million years
  };

  // Generate spiral galaxy structure with natural distribution
  const generateGalaxyParticles = (): StarParticle[] => {
    const particles: StarParticle[] = [];
    
    for (let i = 0; i < galaxyParams.particleCount; i++) {
      // Generate radius with exponential distribution (more particles toward center)
      const u = Math.random();
      const r = galaxyParams.coreRadius + (galaxyParams.diskRadius - galaxyParams.coreRadius) * Math.sqrt(u);
      
      // Random angle for base distribution
      const baseTheta = Math.random() * 2 * Math.PI;
      
      // Create spiral density wave - particles naturally cluster along spiral arms
      const spiralFactor = 2; // Number of spiral arms
      const spiralAngle = spiralFactor * (baseTheta - galaxyParams.armPitch * Math.log(r / galaxyParams.coreRadius));
      
      // Add spiral density enhancement (particles more likely to be in spiral arms)
      const spiralDensity = 0.5 + 0.5 * Math.cos(spiralAngle);
      
      // Skip some particles based on spiral density to create natural arm structure
      if (Math.random() > spiralDensity * 0.7 + 0.3) continue;
      
      // Final position with small random perturbation
      const perturbation = (Math.random() - 0.5) * 20;
      const theta = baseTheta + perturbation / r;
      
      const x = r * Math.cos(theta);
      const y = r * Math.sin(theta);
      
      // Vertical distribution - thinner toward edges
      const scaleHeight = 15 * (1 - r / galaxyParams.diskRadius * 0.5);
      const z = (Math.random() - 0.5) * scaleHeight;
      
      // Simple brightness based on distance from center
      const brightness = Math.exp(-r / (galaxyParams.diskRadius * 0.4)) * (0.3 + Math.random() * 0.7);
      
      particles.push({
        x,
        y,
        z,
        brightness,
        color: 'white', // Simple white color for now
        size: 0.5 + Math.random() * 1.5,
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
