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
      camera.position.z =125;

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
      const particleCount = 3000;
      const particleGeometry = new THREE.BufferGeometry();
      const positions: number[] = [];
      // Add alpha for fading
      const alphas: number[] = [];
      // Add colors for gravity visualization
      const colors: number[] = [];
      const particles: { r: number; theta: number; y: number; radialSpeed: number; angularSpeed: number; orbiting: boolean; phi: number; fading?: boolean; alpha?: number; fastOrbiting?: boolean; targetRadius: number; gravityRandomness: number; strengthRandomness: number; orbitalNoise: number; }[] = [];
      const ORBIT_RADIUS = 160;
      const EPSILON = 0.01;
      const BLACK_HOLE_RADIUS = 10;
      const EVENT_HORIZON = 1.5;
      const GRAVITY_RADIUS = ORBIT_RADIUS * 0.4; // 24 units - only particles within this radius are affected by gravity

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
        const radialSpeed = 4.0 + Math.random() * 4.0; // Fast supernova expansion speed
        const angularSpeed = (Math.random() - 0.5) * 0.0004;
        // Pre-calculate random values to avoid per-frame jittering
        const gravityRandomness = (Math.random() - 0.5) * 0.3; // ±30% variation
        const strengthRandomness = 0.7 + Math.random() * 0.6; // 70% to 130% of base strength
        const orbitalNoise = (Math.random() - 0.5) * 0.0001; // Small random perturbation
        particles.push({ r, theta, y, radialSpeed, angularSpeed, orbiting: false, phi, fading: false, alpha: 1, fastOrbiting: false, targetRadius, gravityRandomness, strengthRandomness, orbitalNoise });
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

      // --- Smooth phase transitions
      let phaseTransition = 0; // 0 = full explosion, 1 = full blackhole
      
      // Start transition to blackhole phase after particles have mostly expanded
      phaseTimer = window.setTimeout(() => {
        phase = 'blackhole';
        // Smooth transition over 2 seconds
        const transitionDuration = 2000;
        const transitionStart = Date.now();
        
        const smoothTransition = () => {
          const elapsed = Date.now() - transitionStart;
          phaseTransition = Math.min(1, elapsed / transitionDuration);
          
          if (phaseTransition < 1) {
            requestAnimationFrame(smoothTransition);
          }
        };
        smoothTransition();
      }, 2000); // Start transition earlier for smoother effect

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
          
          // SINGLE UNIFIED ORBITAL SYSTEM - No conflicts, one calculation
          const particleDistance = Math.max(1, p.r);
          const galaxyBaseSpeed = 0.00002; // Single speed for all
          
          // Distance-based orbital speed (closer = MUCH faster orbiting)
          const baseOrbitalSpeed = galaxyBaseSpeed * Math.pow(ORBIT_RADIUS / particleDistance, 1.5); // Increased from 0.6
          
          // Dramatic orbital acceleration for close particles
          let orbitalMultiplier = 1;
          if (phase === 'blackhole') {
            const orbitalRadius = ORBIT_RADIUS * 0.9; // Larger orbital zone
            if (p.r < orbitalRadius) {
              const orbitalRatio = (orbitalRadius - p.r) / orbitalRadius;
              // Much more dramatic orbital acceleration
              orbitalMultiplier = 1 + Math.pow(orbitalRatio, 1.5) * 50 * phaseTransition;
            }
          }
          
          const currentAngularSpeed = baseOrbitalSpeed * orbitalMultiplier;
          
          // Add organic randomness to orbital motion (pre-calculated to avoid jittering)
          p.theta += currentAngularSpeed + p.orbitalNoise;
          
          // Removed phi randomization to prevent up/down jittering
          // Phi remains constant for smooth orbital motion
          // RADIAL MOVEMENT - Particles expand outward during explosion
          if (phase === 'explosion') {
            if (!p.orbiting) {
              const distToTarget = p.targetRadius - p.r;
              if (distToTarget > EPSILON) {
                const ease = Math.max(0.01, distToTarget / p.targetRadius);
                p.r += p.radialSpeed * ease * 0.4; // Fast supernova expansion
              } else {
                p.r = p.targetRadius;
                p.orbiting = true;
              }
            }
          } else if (phase === 'blackhole') {
            // VERY GRADUAL GRAVITY ACTIVATION - smooth transition from expansion
            const maxInfluenceDistance = ORBIT_RADIUS * 0.6; // Smaller influence zone
            
            // Only apply gravity if particle is within influence range AND black hole is active
            if (p.r < maxInfluenceDistance && !p.fading && phaseTransition > 0.5) {
              // Much more gradual gravity influence
              const gravityInfluence = Math.max(0, 1 - (p.r / maxInfluenceDistance));
              const baseGravityStrength = Math.pow(gravityInfluence, 6) * 0.0003; // Much gentler
              
              // Very gradual activation with delayed start
              const delayedTransition = Math.max(0, (phaseTransition - 0.5) * 2); // Start at 50% transition
              const smoothGravityStrength = baseGravityStrength * delayedTransition * p.strengthRandomness;
              
              // Apply gravitational pull
              if (p.r > BLACK_HOLE_RADIUS) {
                p.r *= (1 - smoothGravityStrength);
              } else {
                // Particle is within black hole radius - rapid spiral and fade
                p.r *= 0.975;
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
          // Use same pre-calculated random gravity radius for visual consistency
          const colorEffectiveGravityRadius = GRAVITY_RADIUS * (1 + p.gravityRandomness);
          if (phase === 'blackhole' && p.r < colorEffectiveGravityRadius) {
            // colorArr[i * 3] = 1;     // Red
            // colorArr[i * 3 + 1] = 0; // Green
            // colorArr[i * 3 + 2] = 0; // Blue
            colorArr[i * 3] = 1;     // White
            colorArr[i * 3 + 1] = 1; // White
            colorArr[i * 3 + 2] = 1; // White
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
