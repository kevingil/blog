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
      camera.position.z =140;

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
        const particles: { r: number; theta: number; y: number; radialSpeed: number; angularSpeed: number; orbiting: boolean; phi: number; fading?: boolean; alpha?: number; fastOrbiting?: boolean; targetRadius: number; gravityRandomness: number; strengthRandomness: number; orbitalNoise: number; angularMomentum: number; orbitalEnergy: number; orbitalTime: number; equatorialPull: number; }[] = [];
      const ORBIT_RADIUS = 160;
      const BLACK_HOLE_RADIUS = 10;
      const EVENT_HORIZON = 1.5;
      const GRAVITY_RADIUS = ORBIT_RADIUS * 0.4; // 24 units - only particles within this radius are affected by gravity

      // Cubic-bezier curve function for gravity effects: cubic-bezier(0, 1.054, 0, 0.941)

      for (let i = 0; i < particleCount; i++) {
        // MAJOR REDESIGN: Particles start with net angular momentum for natural disc formation
        const theta = Math.random() * 2 * Math.PI;
        
        // Start with uniform spherical distribution - let physics create the disc
        const phi = Math.acos(2 * Math.random() - 1);
        
        // All particles start from center (r = 0) for explosion effect
        const r = 0;
        
        // Store target radius for expansion
        const outerThreshold = ORBIT_RADIUS * 0.6;
        const minRadius = 0.3;
        const maxRadius = ORBIT_RADIUS;
        
        let targetRadius: number;
        if (Math.random() < 0.6) {
          const outerBias = Math.pow(Math.random(), 0.3);
          targetRadius = outerThreshold + outerBias * (maxRadius - outerThreshold);
        } else {
          const innerBias = Math.pow(Math.random(), 0.8);
          targetRadius = minRadius + innerBias * (outerThreshold - minRadius);
        }
        
        const x = Math.sin(phi) * Math.cos(theta) * r;
        const y = Math.cos(phi) * r;
        const z = Math.sin(phi) * Math.sin(theta) * r;
        const radialSpeed = 4.0 + Math.random() * 4.0;
        
        // CRITICAL: Give particles significant angular momentum in a preferred direction
        // This creates the net rotation needed for disc formation
        const baseAngularSpeed = 0.002; // Much larger base rotation
        const rotationDirection = 1; // All particles rotate in same direction
        const angularSpeed = baseAngularSpeed * rotationDirection * (0.8 + Math.random() * 0.4); // 80-120% of base
        
        const gravityRandomness = (Math.random() - 0.5) * 0.3;
        const strengthRandomness = 0.7 + Math.random() * 0.6;
        const orbitalNoise = (Math.random() - 0.5) * 0.0001;
        
        // Calculate proper angular momentum for orbital mechanics
        const initialAngularMomentum = targetRadius * angularSpeed;
        const initialOrbitalEnergy = 0.5 * radialSpeed * radialSpeed + initialAngularMomentum * initialAngularMomentum / (2 * targetRadius * targetRadius);
        
        // Initialize orbital tracking for equatorial pull system
        const initialOrbitalTime = 0; // Track how long particle has been orbiting
        const initialEquatorialPull = 0; // Track accumulated equatorial attraction
        
        particles.push({ r, theta, y, radialSpeed, angularSpeed, orbiting: false, phi, fading: false, alpha: 1, fastOrbiting: false, targetRadius, gravityRandomness, strengthRandomness, orbitalNoise, angularMomentum: initialAngularMomentum, orbitalEnergy: initialOrbitalEnergy, orbitalTime: initialOrbitalTime, equatorialPull: initialEquatorialPull });
        positions.push(x, y, z);
        alphas.push(1);
        colors.push(1, 1, 1);
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
          
          // ANGULAR MOMENTUM CONSERVATION - Proper orbital mechanics
          const particleDistance = Math.max(1, p.r);
          
          // Use conservation of angular momentum: L = r * v_tangential
          // As r decreases, v_tangential must increase to conserve L
          let currentAngularSpeed = p.angularMomentum / (particleDistance * particleDistance);
          
          // Add gravitational acceleration for black hole phase
          if (phase === 'blackhole') {
            const gravityInfluenceRadius = ORBIT_RADIUS * 0.8;
            if (p.r < gravityInfluenceRadius) {
              const distanceToEdge = gravityInfluenceRadius - p.r;
              const gravityRatio = distanceToEdge / gravityInfluenceRadius;
              
              // Gradual acceleration buildup
              const accelerationBase = 0.1;
              const maxAcceleration = 50; // Reduced for more realistic motion
              const slowBuildupFactor = Math.pow(gravityRatio, 0.3);
              const phaseInfluence = Math.pow(phaseTransition, 2);
              const finalAcceleration = accelerationBase + (maxAcceleration - accelerationBase) * slowBuildupFactor * phaseInfluence;
              
              currentAngularSpeed *= (1 + finalAcceleration * 0.1); // More moderate increase
            }
          }
          
          // Add organic randomness to orbital motion (pre-calculated to avoid jittering)
          p.theta += currentAngularSpeed + p.orbitalNoise;
          
          // Removed phi randomization to prevent up/down jittering
          // Phi remains constant for smooth orbital motion
          // RADIAL MOVEMENT - Particles expand outward during explosion with asymptotic slowdown
          if (phase === 'explosion') {
            const distToTarget = p.targetRadius - p.r;
            
            // Only expand if we haven't reached the target
            if (distToTarget > 0) {
              // Asymptotic approach - speed proportional to remaining distance
              // This creates natural slowdown that approaches but never reaches the target
              const distanceRatio = distToTarget / p.targetRadius;
              
              // Asymptotic curve: fast when far, extremely slow when close
              // Uses a power curve that ensures we never overshoot
              const slowdownFactor = Math.pow(distanceRatio, 0.5); // Square root for gentler approach
              
              // Calculate expansion step that's proportional to remaining distance
              const expansionStep = distToTarget * 0.05 * slowdownFactor; // 5% of remaining distance
              
              // Ensure we never overshoot the target
              const safeExpansion = Math.min(expansionStep, distToTarget * 0.99);
              
              p.r += safeExpansion;
              
              // Mark as orbiting once we get very close (but not at target)
              if (distToTarget < 0.1) {
                p.orbiting = true;
              }
            }
          } else if (phase === 'blackhole') {
            // ASYMPTOTIC GRAVITY ACTIVATION - gradual buildup like expansion
            const maxInfluenceDistance = ORBIT_RADIUS * 0.8; // Larger influence zone for smoother transition
            
            // Only apply gravity if particle is within influence range AND black hole is active
            if (p.r < maxInfluenceDistance && !p.fading && phaseTransition > 0.1) {
              const distanceToEdge = maxInfluenceDistance - p.r;
              const distanceRatio = distanceToEdge / maxInfluenceDistance;
              
              // Much gentler asymptotic gravity buildup
              const gravityBase = 0.000005; // Much weaker minimum gravity
              const maxGravityStrength = 0.0003; // Reduced maximum gravity strength
              
              // Even gentler asymptotic approach
              const gravityBuildupFactor = Math.pow(distanceRatio, 0.6); // More gentle buildup
              
              // Much more gradual phase activation
              const phaseInfluence = Math.pow(Math.max(0, (phaseTransition - 0.1) / 0.9), 4); // Quartic buildup - very slow
              
              // Calculate final gravity strength with asymptotic buildup
              const baseGravityStrength = gravityBase + (maxGravityStrength - gravityBase) * gravityBuildupFactor * phaseInfluence;
              const finalGravityStrength = baseGravityStrength * p.strengthRandomness;
              
              // Active black hole physics - disc formation and polar jets
              if (p.r > BLACK_HOLE_RADIUS) {
                // Calculate particle's position relative to equatorial plane
                const particleHeight = Math.abs(Math.cos(p.phi)); // 0 = equator, 1 = poles
                const equatorialFactor = 1 - particleHeight; // 1 = equator, 0 = poles
                
                                 // No artificial disc preference - let natural orbital mechanics work
                 const discGravity = finalGravityStrength; // Same gravity everywhere
                
                                                  // Natural disc formation through orbital mechanics
                 const pullDistance = p.r - BLACK_HOLE_RADIUS;
                 const safePull = pullDistance * discGravity;
                 const maxPull = pullDistance * 0.05; // Even gentler max pull (5%)
                 
                 p.r -= Math.min(safePull, maxPull);
                 
                                  // ORBITAL-SPEED BASED DISC FORMATION - Faster orbiting = stronger equatorial pull
                 if (p.r < maxInfluenceDistance && phase === 'blackhole') {
                   // Mark as orbiting so particles can accumulate orbital activity
                   p.orbiting = true;
                   
                   // Accumulate orbital time based on current angular speed - much slower
                   p.orbitalTime += currentAngularSpeed * 20; // Much slower accumulation
                   
                   // Define equatorial plane
                   const EQUATOR = Math.PI / 2;
                   
                   // Calculate how far particle is from equator
                   const distanceFromEquator = Math.abs(p.phi - EQUATOR);
                   
                   // ORBITAL-SPEED BASED FLATTENING - MUCH SLOWER
                   if (distanceFromEquator > 0.01) { // Only if not already at equator
                     // Flattening strength based on:
                     // 1. Current orbital speed (faster orbiting = stronger pull)
                     // 2. Accumulated orbital time (more orbiting = stronger pull)
                     // 3. Distance from equator (farther = stronger pull)
                     
                     const orbitalSpeedFactor = currentAngularSpeed * 500; // Much smaller factor
                     const orbitalTimeFactor = Math.min(p.orbitalTime / 1000, 1); // Takes 10x longer to mature
                     const distanceFactor = distanceFromEquator / (Math.PI / 2); // Distance from equator (0-1)
                     
                     // Combine all factors for natural, progressive flattening - much weaker
                     const flatteningStrength = orbitalSpeedFactor * orbitalTimeFactor * distanceFactor * 0.001; // 20x weaker
                     
                     // Pull directly toward equatorial plane
                     if (p.phi > EQUATOR) {
                       p.phi -= flatteningStrength; // Pull from above
                     } else {
                       p.phi += flatteningStrength; // Pull from below
                     }
                     
                     // Ensure we don't overshoot
                     if (Math.abs(p.phi - EQUATOR) < flatteningStrength) {
                       p.phi = EQUATOR;
                     }
                   }
                 }
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
