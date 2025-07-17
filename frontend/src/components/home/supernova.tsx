import React, { useRef, useEffect } from 'react';

// Utility to dynamically load Three.js from CDN
function loadThreeJS(): Promise<any> {
  return new Promise((resolve, reject) => {
    // @ts-ignore
    if ((window as any).THREE) return resolve((window as any).THREE);
    const script = document.createElement('script');
    script.src = 'https://unpkg.com/three@0.153.0/build/three.min.js';
    script.onload = () => resolve((window as any).THREE);
    script.onerror = reject;
    document.body.appendChild(script);
  });
}

export const SupernovaAnimation: React.FC<{ zIndex?: number }> = ({ zIndex = -1 }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animationRef = useRef<number>();

  useEffect(() => {
    let renderer: any, scene: any, camera: any, quasar: any, blackHole: any;
    let width = window.innerWidth;
    let height = window.innerHeight;
    let frameId: number;
    let THREE: any;
    let phase: 'explosion' | 'blackhole' = 'explosion';
    let phaseTimer: number;
    let gravityTimer: number;

    let cleanup = () => {};

    loadThreeJS().then((_THREE: any) => {
      THREE = _THREE;
      // Renderer
      renderer = new THREE.WebGLRenderer({ canvas: canvasRef.current, alpha: true, antialias: true });
      renderer.setSize(width, height);
      // Scene
      scene = new THREE.Scene();
      // Camera
      camera = new THREE.PerspectiveCamera(75, width / height, 0.1, 1000);
      camera.position.z =150;

      // --- Quasar (center of gravity) ---
      // Placeholder: bright sphere, ready for shader
      const quasarGeometry = new THREE.SphereGeometry(2, 32, 32);
      const quasarMaterial = new THREE.MeshBasicMaterial({ color: 0xffffcc });
      quasar = new THREE.Mesh(quasarGeometry, quasarMaterial);
      scene.add(quasar);

      // --- Black hole (hidden at first) ---
      const blackHoleGeometry = new THREE.SphereGeometry(1.5, 32, 32);
      const blackHoleMaterial = new THREE.MeshBasicMaterial({ color: 0x000000 });
      blackHole = new THREE.Mesh(blackHoleGeometry, blackHoleMaterial);
      blackHole.visible = false;
      scene.add(blackHole);

      // --- Particles (explosion + orbit) ---
      const particleCount = 6000;
      const particleGeometry = new THREE.BufferGeometry();
      const positions: number[] = [];
      // Add alpha for fading
      const alphas: number[] = [];
      // Add colors for gravity visualization
      const colors: number[] = [];
      const particles: { r: number; theta: number; y: number; radialSpeed: number; angularSpeed: number; orbiting: boolean; phi: number; fading?: boolean; alpha?: number; fastOrbiting?: boolean; targetRadius?: number; }[] = [];
      const ORBIT_RADIUS = 120;
      const EPSILON = 0.01;
      const BLACK_HOLE_RADIUS = 10;
      const EVENT_HORIZON = 1.5;
      const GRAVITY_RADIUS = ORBIT_RADIUS * 0.7; // 24 units - only particles within this radius are affected by gravity

      // Cubic-bezier curve function for gravity effects: cubic-bezier(0, 1.054, 0, 0.941)
      function cubicBezier(t: number): number {
        // Cubic-bezier curve approximation for (0, 1.054, 0, 0.941)
        // This creates a smooth easing curve
        const p1 = 1.054;
        const p2 = 0.941;
        const u = 1 - t;
        return 3 * u * u * t * p1 + 3 * u * t * t * p2 + t * t * t;
      }

      for (let i = 0; i < particleCount; i++) {
        const theta = Math.random() * 2 * Math.PI;
        const phi = Math.acos(2 * Math.random() - 1);
        
        // All particles start from center (r = 0) for explosion effect
        const r = 0;
        
        // Store target radius for expansion - maintains original distribution
        const outerThreshold = ORBIT_RADIUS * 0.6; // 72 units
        const minRadius = 0.3;
        const maxRadius = ORBIT_RADIUS;
        
        let targetRadius: number;
        if (Math.random() < 0.6) {
          // 60% of particles: expand to outer threshold to max radius
          const outerBias = Math.pow(Math.random(), 0.3); // Bias towards outer edge
          targetRadius = outerThreshold + outerBias * (maxRadius - outerThreshold);
        } else {
          // 40% of particles: expand to min radius to outer threshold
          const innerBias = Math.pow(Math.random(), 0.8); // Less bias, more uniform
          targetRadius = minRadius + innerBias * (outerThreshold - minRadius);
        }
        
        const x = Math.sin(phi) * Math.cos(theta) * r;
        const y = Math.cos(phi) * r;
        const z = Math.sin(phi) * Math.sin(theta) * r;
        const radialSpeed = 2.5 + Math.random() * 2.5;
        const angularSpeed = (Math.random() - 0.5) * 0.0004;
        particles.push({ r, theta, y, radialSpeed, angularSpeed, orbiting: false, phi, fading: false, alpha: 1, fastOrbiting: false, targetRadius });
        positions.push(x, y, z);
        alphas.push(1);
        colors.push(1, 1, 1); // Start with white color (RGB)
      }
      particleGeometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
      // Add alpha attribute for fading
      particleGeometry.setAttribute('alpha', new THREE.Float32BufferAttribute(alphas, 1));
      // Add color attribute for gravity visualization
      particleGeometry.setAttribute('color', new THREE.Float32BufferAttribute(colors, 3));
      // Use vertex colors for gravity visualization
      const particleMaterial = new THREE.PointsMaterial({ size: 0.3, transparent: true, vertexColors: true });
      const particleSystem = new THREE.Points(particleGeometry, particleMaterial);
      scene.add(particleSystem);

      // --- Phase timer slower because "phase" dictates the expansion of the particles
      phaseTimer = window.setTimeout(() => {
        phase = 'blackhole';
      }, 3000);

      gravityTimer = window.setTimeout(() => {
        quasar.visible = false;
        blackHole.visible = true;
      }, 1000);

      // --- Animation loop ---
      function animate() {
        const pos = particleGeometry.attributes.position.array as number[];
        const alphaArr = particleGeometry.attributes.alpha.array as number[];
        const colorArr = particleGeometry.attributes.color.array as number[];
        for (let i = 0; i < particleCount; i++) {
          let p = particles[i];
          // Galaxy-like rotation: farther particles rotate slower (like a spiral galaxy)
          // This creates a more realistic galactic rotation curve
          const baseRotationSpeed = 0.0003; // Much slower base rotation for visible spiral effect
          const distanceRotationFactor = ORBIT_RADIUS / (p.r + 5); // Farther = slower (galaxy-like)
          
          // 3x faster orbital speed for particles being sucked in by black hole
          let orbitalSpeedMultiplier = 1;
          if (phase === 'blackhole' && p.r < GRAVITY_RADIUS && !p.fading) {
            orbitalSpeedMultiplier = 3; // 3x faster orbital speed when being sucked in
          }
          
          const currentAngularSpeed = baseRotationSpeed * distanceRotationFactor * orbitalSpeedMultiplier;
          p.theta += currentAngularSpeed;
          // Particles expand outward but don't all reach the same final radius
          if (phase === 'explosion') {
            if (!p.orbiting) {
              // Each particle has its own target radius based on initial distribution
              if (!p.targetRadius) {
                // Set target radius based on initial spawn position and some expansion factor
                p.targetRadius = p.r * (8 + Math.random() * 12); // Expand by 8-20x original radius
                p.targetRadius = Math.min(p.targetRadius, ORBIT_RADIUS); // Cap at orbit radius
              }
              
              const distToTarget = p.targetRadius - p.r;
              if (distToTarget > EPSILON) {
                const ease = Math.max(0.01, distToTarget / p.targetRadius);
                p.r += p.radialSpeed * ease * 0.25; // Halved from 0.5
              } else {
                p.r = p.targetRadius;
                p.orbiting = true;
              }
            }
          } else if (phase === 'blackhole') {
            // In spherical coordinates, p.r is the distance from center
            // Only affect particles within the gravity radius (1/5th of orbit radius = 24 units)
            if (p.r < GRAVITY_RADIUS && !p.fading) {
              // Calculate distance-based gravity strength with smooth falloff
              // Particles at the edge (24 units) have minimal pull, particles closer have stronger pull
              const distanceFromEdge = GRAVITY_RADIUS - p.r;
              const normalizedDistance = distanceFromEdge / GRAVITY_RADIUS; // 0 at edge, 1 at center
              
              // Apply cubic-bezier curve for smooth gravity effects
              const gravityStrength = cubicBezier(normalizedDistance);
              
              // Apply gravitational pull
              if (p.r > BLACK_HOLE_RADIUS) {
                // Pull particle inward with cubic-bezier curve strength (gravity halved again)
                p.r *= (1 - gravityStrength * 0.00375);
                // Angular speed is now automatically handled by distance-based rotation
              } else {
                // Particle is within black hole radius - rapid spiral and fade
                p.r *= 0.975; // Slower fade (was 0.95)
                if (p.r < EVENT_HORIZON) {
                  p.fading = true;
                }
              }
            }
            
            // Handle fading particles
            if (p.fading) {
              p.alpha = Math.max(0, (p.alpha ?? 1) - 0.03);
              if (p.alpha === 0) {
                p.r = 0;
              }
            }
          }
          // Spherical to Cartesian for position
          const x = Math.sin(p.phi) * Math.cos(p.theta) * p.r;
          const y = Math.cos(p.phi) * p.r;
          const z = Math.sin(p.phi) * Math.sin(p.theta) * p.r;
          
          pos[i * 3] = x;
          pos[i * 3 + 1] = y;
          pos[i * 3 + 2] = z;
          alphaArr[i] = p.alpha ?? 1;
          
          // Color particles red if they're within gravity radius during black hole phase
          if (phase === 'blackhole' && p.r < GRAVITY_RADIUS) {
            colorArr[i * 3] = 1;     // Red
            colorArr[i * 3 + 1] = 0; // Green
            colorArr[i * 3 + 2] = 0; // Blue
          } else {
            colorArr[i * 3] = 1;     // White
            colorArr[i * 3 + 1] = 1; // White
            colorArr[i * 3 + 2] = 1; // White
          }
        }
        particleGeometry.attributes.position.needsUpdate = true;
        particleGeometry.attributes.alpha.needsUpdate = true;
        particleGeometry.attributes.color.needsUpdate = true;
        renderer.render(scene, camera);
        frameId = requestAnimationFrame(animate);
      }
      animate();
      animationRef.current = frameId;

      // --- Resize handler ---
      function handleResize() {
        width = window.innerWidth;
        height = window.innerHeight;
        renderer.setSize(width, height);
        camera.aspect = width / height;
        camera.updateProjectionMatrix();
      }
      window.addEventListener('resize', handleResize);

      // --- Cleanup ---
      cleanup = () => {
        cancelAnimationFrame(frameId);
        window.removeEventListener('resize', handleResize);
        clearTimeout(phaseTimer);
        renderer.dispose();
        particleGeometry.dispose();
        particleMaterial.dispose();
        quasarGeometry.dispose();
        quasarMaterial.dispose();
        blackHoleGeometry.dispose();
        blackHoleMaterial.dispose();
        scene.clear();
      };
    });
    return () => cleanup();
  }, []);

  return (
    <div
      className="supernova-animation"
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        width: '100vw',
        height: '100vh',
        zIndex,
        pointerEvents: 'none', // allow clicks through
        overflow: 'hidden',
      }}
    >
      <canvas
        ref={canvasRef}
        style={{ width: '100%', height: '100%', display: 'block', pointerEvents: 'none' }}
        tabIndex={-1}
        aria-hidden="true"
      />
    </div>
  );
};

export default SupernovaAnimation; 
