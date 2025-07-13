import React, { useRef, useEffect } from 'react';
import * as THREE from 'three';

interface SupernovaProps {
  className?: string;
}

/**
 * SupernovaAnimation
 *
 * A realistic supernova simulation with particles that expand outward,
 * some getting captured in orbit around a central black hole, creating
 * a dynamic accretion disk and nebula effect.
 */
export const SupernovaAnimation: React.FC<SupernovaProps> = ({ className = '' }) => {
  const mountRef = useRef<HTMLDivElement>(null);
  const animationRef = useRef<number>();

  useEffect(() => {
    if (!mountRef.current) return;

    /* ------------------------------------------------------------
     * Scene / Camera / Renderer
     * ---------------------------------------------------------- */
    const scene = new THREE.Scene();

    const camera = new THREE.PerspectiveCamera(
      65,
      window.innerWidth / window.innerHeight,
      0.1,
      1200
    );
    camera.position.set(0, 0, 12);
    camera.lookAt(new THREE.Vector3(0, 0, 0));

    const renderer = new THREE.WebGLRenderer({ alpha: true, antialias: false });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setClearColor(0x000000, 0);
    mountRef.current.appendChild(renderer.domElement);

    /* ------------------------------------------------------------
     * Particle Buffer Setup with Orbital Physics
     * ---------------------------------------------------------- */
    const PARTICLE_COUNT = 40000;
    const initPositions = new Float32Array(PARTICLE_COUNT * 3);
    const velocities = new Float32Array(PARTICLE_COUNT * 3);
    const masses = new Float32Array(PARTICLE_COUNT);
    const colors = new Float32Array(PARTICLE_COUNT * 3);
    const lifetimes = new Float32Array(PARTICLE_COUNT);

    // Black hole parameters - realistic large-scale gravity
    const BLACK_HOLE_MASS = 25.0; // Stronger for large scale effects
    const SCHWARZSCHILD_RADIUS = 0.8; // Event horizon
    const ACCRETION_DISK_RADIUS = 8.0; // Much larger influence zone for orbital mechanics
    const BLACK_HOLE_FORMATION_DELAY = 2.5; // Earlier formation - was 4.0
    const BLACK_HOLE_FORMATION_DURATION = 2.0; // Faster formation - was 3.0

    for (let i = 0; i < PARTICLE_COUNT; i++) {
      const i3 = i * 3;

      // Initial explosion from center with proper 3D distribution
      const explosionRadius = THREE.MathUtils.randFloat(0.08, 0.6); // Slightly larger minimum
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(THREE.MathUtils.randFloatSpread(2));
      
      initPositions[i3] = explosionRadius * Math.sin(phi) * Math.cos(theta);
      initPositions[i3 + 1] = explosionRadius * Math.sin(phi) * Math.sin(theta);
      initPositions[i3 + 2] = explosionRadius * Math.cos(phi);

      // Velocity distribution that creates natural particle stopping
      const distanceFromCenter = explosionRadius;
      
      // Base speed varies significantly - creates natural separation
      const baseSpeed = THREE.MathUtils.randFloat(1.2, 6.5); // Slightly increased range
      
      // Particles at different distances get different speed profiles
      let speedMultiplier;
      if (distanceFromCenter < 0.10) {
        // Very inner particles: extremely low speeds (only these can be captured)
        speedMultiplier = THREE.MathUtils.randFloat(0.2, 0.5);
      } else if (distanceFromCenter < 0.25) {
        // Middle particles: medium speeds (will stop at medium distance)
        speedMultiplier = THREE.MathUtils.randFloat(0.6, 1.1);
      } else {
        // Outer particles: higher speeds (form outer nebula, immune to black hole)
        speedMultiplier = THREE.MathUtils.randFloat(0.9, 1.4);
      }
      
      const speed = baseSpeed * speedMultiplier;
      
      const velocityDir = new THREE.Vector3(
        initPositions[i3],
        initPositions[i3 + 1], 
        initPositions[i3 + 2]
      ).normalize();

      // Reduced angular momentum for more radial expansion
      const tangent = new THREE.Vector3(-velocityDir.y, velocityDir.x, 0).normalize();
      const orbitalComponent = THREE.MathUtils.randFloat(0.0, 0.3) * (distanceFromCenter < 0.2 ? 1.0 : 0.5);
      
      velocities[i3] = velocityDir.x * speed + tangent.x * orbitalComponent;
      velocities[i3 + 1] = velocityDir.y * speed + tangent.y * orbitalComponent;
      velocities[i3 + 2] = velocityDir.z * speed + tangent.z * orbitalComponent * 0.4;

      masses[i] = THREE.MathUtils.randFloat(0.3, 1.0);
      lifetimes[i] = THREE.MathUtils.randFloat(0.0, 0.6); // Quicker birth stagger

      // Color based on initial speed and position
      const normalizedSpeed = (speed - 1.5) / 6.5;
      const positionHeat = (1.0 - distanceFromCenter / 0.5) * 0.3;
      const heatFactor = normalizedSpeed + positionHeat;
      
      colors[i3] = Math.min(1.0, 0.5 + heatFactor * 0.7); // Red
      colors[i3 + 1] = Math.min(1.0, 0.1 + heatFactor * 0.8); // Green  
      colors[i3 + 2] = Math.min(1.0, 0.05 + heatFactor * 0.95); // Blue
    }

    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute('aInitPos', new THREE.BufferAttribute(initPositions, 3));
    geometry.setAttribute('aVelocity', new THREE.BufferAttribute(velocities, 3));
    geometry.setAttribute('aMass', new THREE.BufferAttribute(masses, 1));
    geometry.setAttribute('aColor', new THREE.BufferAttribute(colors, 3));
    geometry.setAttribute('aLifetime', new THREE.BufferAttribute(lifetimes, 1));
    geometry.setAttribute('position', new THREE.BufferAttribute(initPositions, 3));

    /* ------------------------------------------------------------
     * Advanced Physics Shader
     * ---------------------------------------------------------- */
    const vertexShader = /* glsl */`
      precision highp float;

      attribute vec3 aInitPos;
      attribute vec3 aVelocity;
      attribute float aMass;
      attribute vec3 aColor;
      attribute float aLifetime;

      uniform float uTime;
      uniform float uBlackHoleMass;
      uniform float uSchwarzschildRadius;
      uniform float uAccretionRadius;
      uniform float uDamping;
      uniform float uBlackHoleFormationDelay;
      uniform float uBlackHoleFormationDuration;

      varying vec3 vColor;
      varying float vDistanceFromBH;
      varying float vVelocityMag;
      varying float vAge;
      varying float vBlackHoleStrength;
         
         void main() {
        float age = max(uTime - aLifetime, 0.0);
        vAge = age;
        
        if (age <= 0.0) {
          gl_Position = vec4(0.0, 0.0, -1000.0, 1.0);
          return;
        }

        // Calculate black hole formation progress
        float blackHoleAge = max(uTime - uBlackHoleFormationDelay, 0.0);
        float blackHoleStrength = smoothstep(0.0, uBlackHoleFormationDuration, blackHoleAge);
        vBlackHoleStrength = blackHoleStrength;

        // Calculate particle expansion - TRULY STOP at maximum distance
        vec3 baseVelocity = aVelocity;
        float initialSpeed = length(baseVelocity);
        vec3 expansionDir = normalize(baseVelocity);
        
        // Each particle has a definitive maximum expansion distance
        float maxExpansionDistance = initialSpeed * 1.8; // Reduced from 2.0
        
        // Exponential approach to maximum distance - particles STOP
        float timeConstant = 3.0 + (maxExpansionDistance / 8.0); // Slower for farther particles
        float expansionProgress = 1.0 - exp(-age / timeConstant);
        
        // Ensure particles actually reach and STAY at maximum distance
        float currentExpansionDist = maxExpansionDistance * min(expansionProgress, 0.98);
        
        // Base position from expansion (no more movement after reaching max)
        vec3 pos = aInitPos + expansionDir * currentExpansionDist;
        
        // MUCH smaller wobbling - only when fully stopped
        if (expansionProgress > 0.90) {
          float wobbleIntensity = (expansionProgress - 0.90) * 2.0; // Much reduced
          
          // Very gentle wobbling motion
          vec3 wobble = vec3(
            sin(uTime * 0.3 + length(aInitPos) * 3.0) * 0.08,  // Much smaller amplitude
            cos(uTime * 0.25 + length(aInitPos) * 2.5) * 0.08,
            sin(uTime * 0.2 + length(aInitPos) * 2.0) * 0.05
          ) * wobbleIntensity * min(maxExpansionDistance / 15.0, 1.0); // Scale with distance
          
          pos += wobble;
        }
        
        // Black hole influence - proper gravitational physics
        if (blackHoleStrength > 0.0) {
          float distanceFromCenter = length(pos);
          
          // Large influence zone for realistic orbital mechanics
          float maxInfluenceRadius = uAccretionRadius * 1.2; // Much larger reach
          
          if (distanceFromCenter < maxInfluenceRadius) {
            
            // Proper inverse square law gravity (F = GM/r²)
            float gravityStrength = (uBlackHoleMass * blackHoleStrength) / (distanceFromCenter * distanceFromCenter + 0.1);
            
            // Distance-based influence falloff for large scale
            float distanceFalloff = 1.0 / (1.0 + distanceFromCenter * 0.3); // Gradual falloff
            gravityStrength *= distanceFalloff;
            
            // Only apply if gravity is strong enough to matter
            if (gravityStrength > 0.05) {
              
              vec3 dirToCenter = -normalize(pos);
              
              // Calculate gravitational acceleration
              vec3 gravityAccel = dirToCenter * gravityStrength / aMass;
              
              // Apply gravitational pull over time
              float gravityTime = blackHoleAge;
              vec3 gravityDisplacement = gravityAccel * gravityTime * gravityTime * 0.5;
              
              // Different behavior based on distance
              if (distanceFromCenter > uAccretionRadius * 0.6) {
                // Far particles: Large stable orbits with slow decay
                
                // Calculate orbital motion
                float orbitalSpeed = sqrt(gravityStrength * distanceFromCenter) * 0.8;
                float orbitalAngle = orbitalSpeed * gravityTime / distanceFromCenter;
                
                // Apply orbital rotation
                vec3 orbitPos = pos;
                pos.x = orbitPos.x * cos(orbitalAngle) - orbitPos.y * sin(orbitalAngle);
                pos.y = orbitPos.x * sin(orbitalAngle) + orbitPos.y * cos(orbitalAngle);
                
                // Very slow orbital decay for distant particles
                float decayRate = 1.0 - (gravityStrength * 0.008);
                pos *= max(decayRate, 0.85); // Don't decay too fast
                
              } else if (distanceFromCenter > uSchwarzschildRadius * 3.0) {
                // Medium particles: Closer orbits with faster decay
                
                // Calculate orbital motion
                float orbitalSpeed = sqrt(gravityStrength * distanceFromCenter) * 1.0;
                float orbitalAngle = orbitalSpeed * gravityTime / distanceFromCenter;
                
                // Apply orbital rotation
                vec3 orbitPos = pos;
                pos.x = orbitPos.x * cos(orbitalAngle) - orbitPos.y * sin(orbitalAngle);
                pos.y = orbitPos.x * sin(orbitalAngle) + orbitPos.y * cos(orbitalAngle);
                
                // Moderate orbital decay
                float decayRate = 1.0 - (gravityStrength * 0.015);
                pos *= max(decayRate, 0.7);
                
                // Additional inward pull
                pos += gravityDisplacement * 0.3;
                
              } else {
                // Close particles: Direct consumption with minimal orbit
                pos += gravityDisplacement;
                
                // Small orbital motion even when being consumed
                if (distanceFromCenter > uSchwarzschildRadius * 1.5) {
                  float orbitalSpeed = sqrt(gravityStrength * distanceFromCenter) * 0.5;
                  float orbitalAngle = orbitalSpeed * gravityTime / distanceFromCenter;
                  
                  vec3 orbitPos = pos;
                  pos.x = orbitPos.x * cos(orbitalAngle) - orbitPos.y * sin(orbitalAngle);
                  pos.y = orbitPos.x * sin(orbitalAngle) + orbitPos.y * cos(orbitalAngle);
                }
              }
            }
          }
        }

        // Consume particles that get too close
        if (length(pos) < uSchwarzschildRadius * 0.8 && blackHoleStrength > 0.8) {
          pos = vec3(0.0, 0.0, 0.0);
        }

        vDistanceFromBH = length(pos);
        
        // Calculate velocity for visual effects (should be very low for stopped particles)
        vec3 prevPos = aInitPos + expansionDir * (maxExpansionDistance * min(1.0 - exp(-(age - 0.1) / timeConstant), 0.98));
        vVelocityMag = length((pos - prevPos) / 0.1) * 0.3; // Reduce apparent velocity
        
        // Color based on distance and very subtle motion
        float temperatureFactor = vVelocityMag * 0.05 + (1.0 / (vDistanceFromBH + 2.0)) * blackHoleStrength * 0.3;
        vColor = aColor * (0.25 + temperatureFactor * 0.6); // Dimmer overall
        
        gl_Position = projectionMatrix * modelViewMatrix * vec4(pos, 1.0);

        // Much smaller particle sizes overall
        float proximityGlow = (uAccretionRadius / (vDistanceFromBH + 2.0)) * blackHoleStrength;
        float baseSize = 1.2 + aMass * 0.8 + proximityGlow * 1.5; // Smaller base size
        
        // Gravitational lensing for particles behind black hole
        vec4 screenPos = projectionMatrix * modelViewMatrix * vec4(pos, 1.0);
        vec4 bhScreenPos = projectionMatrix * modelViewMatrix * vec4(0.0, 0.0, 0.0, 1.0);
        float screenDistance = length(screenPos.xy - bhScreenPos.xy);
        
        // Subtle lensing effect
        if (screenDistance < 0.2 && screenDistance > 0.05 && pos.z < -0.5) {
          baseSize *= (1.0 + (0.2 - screenDistance) * 2.0);
        }
        
        float distance = length((modelViewMatrix * vec4(pos, 1.0)).xyz);
        gl_PointSize = baseSize * (30.0 / distance); // Smaller overall
      }
    `;

    const fragmentShader = /* glsl */`
      precision highp float;

      uniform float uTime;
      uniform float uSchwarzschildRadius;
      uniform float uAccretionRadius;
      uniform float uBlackHoleFormationDelay;

      varying vec3 vColor;
      varying float vDistanceFromBH;
      varying float vVelocityMag;
      varying float vAge;
      varying float vBlackHoleStrength;
         
         void main() {
        // Circular particle shape with soft edges
        vec2 uv = gl_PointCoord * 2.0 - 1.0;
        float dist = length(uv);
        
        if (dist > 1.0) discard;
        
        // Soft circular falloff
        float alpha = 1.0 - smoothstep(0.2, 1.0, dist);
        
        // Intensity based on velocity and black hole proximity
        float baseIntensity = 0.3 + vVelocityMag * 0.06;
        float accretionGlow = smoothstep(uAccretionRadius * 2.0, uAccretionRadius * 0.5, vDistanceFromBH);
        baseIntensity += accretionGlow * 0.7 * vBlackHoleStrength;
        
        // Color modulation - particles get hotter near the black hole
        vec3 finalColor = vColor * baseIntensity;
        
        // Add blue-white core for high-energy particles accelerated by black hole
        if (vVelocityMag > 4.0 && vBlackHoleStrength > 0.3) {
          float coreAlpha = smoothstep(0.7, 0.2, dist);
          float energyBoost = vBlackHoleStrength * accretionGlow;
          finalColor = mix(finalColor, vec3(0.9, 0.95, 1.0), coreAlpha * energyBoost * 0.6);
        }
        
        // NO aging fade - particles should remain visible indefinitely
        // Removed: float ageFade = 1.0 - smoothstep(20.0, 30.0, vAge);
        
        // Boost alpha for particles being accelerated by black hole
        alpha *= (1.0 + accretionGlow * vBlackHoleStrength * 0.5);
        
        // Ensure minimum visibility for all particles
        alpha = max(alpha * 0.9, 0.2); // Never go below 20% opacity
        
        gl_FragColor = vec4(finalColor, alpha);
      }
    `;

    const material = new THREE.ShaderMaterial({
      uniforms: {
        uTime: { value: 0 },
        uBlackHoleMass: { value: BLACK_HOLE_MASS },
        uSchwarzschildRadius: { value: SCHWARZSCHILD_RADIUS },
        uAccretionRadius: { value: ACCRETION_DISK_RADIUS },
        uDamping: { value: 0.002 }, // Very light damping for stability
        uBlackHoleFormationDelay: { value: BLACK_HOLE_FORMATION_DELAY },
        uBlackHoleFormationDuration: { value: BLACK_HOLE_FORMATION_DURATION }
      },
      vertexShader,
      fragmentShader,
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending // Better blending for glowing particles
    });

    const points = new THREE.Points(geometry, material);
    scene.add(points);

    /* ------------------------------------------------------------
     * Black Hole Visualization - Forms Gradually
     * ---------------------------------------------------------- */
    const bhGeometry = new THREE.RingGeometry(SCHWARZSCHILD_RADIUS * 0.4, SCHWARZSCHILD_RADIUS * 3.5, 64);

    const bhVertexShader = /* glsl */`
      varying vec2 vUv;
      varying float vRadius;
      
      void main() {
        vUv = uv;
        vRadius = length(position);
        gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
      }
    `;

    const bhFragmentShader = /* glsl */`
      precision highp float;
      
      uniform float uTime;
      uniform float uSchwarzschildRadius;
      uniform float uBlackHoleFormationDelay;
      uniform float uBlackHoleFormationDuration;
      
      varying vec2 vUv;
      varying float vRadius;
      
      void main() {
        // Calculate black hole formation progress
        float blackHoleAge = max(uTime - uBlackHoleFormationDelay, 0.0);
        float formationProgress = smoothstep(0.0, uBlackHoleFormationDuration, blackHoleAge);
        
        // Don't render before formation begins
        if (formationProgress <= 0.0) {
          discard;
        }
        
        float normalizedRadius = vRadius / (uSchwarzschildRadius * 3.5); // Updated for larger geometry
        
        // Create accretion disk rings that intensify as black hole forms
        float rings = sin(normalizedRadius * 8.0 + uTime * 1.2) * 0.5 + 0.5;
        rings *= sin(normalizedRadius * 5.0 - uTime * 0.8) * 0.5 + 0.5;
        
        // Intensity falls off with distance and grows with formation
        float intensity = (1.0 - smoothstep(0.1, 1.0, normalizedRadius)) * formationProgress;
        intensity *= rings * 0.6 + 0.4;
        
        // Orange-red accretion disk color
        vec3 diskColor = vec3(1.0, 0.5, 0.2) * intensity * formationProgress;
        
        // Add gravitational lensing effect
        float lensing = exp(-pow(normalizedRadius * 1.2, 2.0)) * 0.5 * formationProgress;
        diskColor += vec3(0.7, 0.8, 1.0) * lensing;
        
        // Central black hole - completely black when formed
        if (vRadius < uSchwarzschildRadius * 0.6 && formationProgress > 0.7) {
          gl_FragColor = vec4(0.0, 0.0, 0.0, formationProgress);
        } else {
          float alpha = intensity * (1.0 - smoothstep(0.7, 1.0, normalizedRadius)) * formationProgress * 0.8;
          gl_FragColor = vec4(diskColor, alpha);
        }
      }
    `;

    const bhMaterial = new THREE.ShaderMaterial({
      uniforms: {
        uTime: { value: 0 },
        uSchwarzschildRadius: { value: SCHWARZSCHILD_RADIUS },
        uBlackHoleFormationDelay: { value: BLACK_HOLE_FORMATION_DELAY },
        uBlackHoleFormationDuration: { value: BLACK_HOLE_FORMATION_DURATION }
      },
      vertexShader: bhVertexShader,
      fragmentShader: bhFragmentShader,
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending
    });

    const blackHole = new THREE.Mesh(bhGeometry, bhMaterial);
    scene.add(blackHole);

    /* ------------------------------------------------------------
     * Animation Loop
     * ---------------------------------------------------------- */
    const startTime = performance.now();

    const onResize = () => {
      camera.aspect = window.innerWidth / window.innerHeight;
      camera.updateProjectionMatrix();
      renderer.setSize(window.innerWidth, window.innerHeight);
    };
    window.addEventListener('resize', onResize);

    const animate = () => {
      const elapsed = (performance.now() - startTime) / 1000;
      
      // Speed up time for faster black hole formation
      const acceleratedTime = elapsed * 0.6; // Increased from 0.4 to 0.6
      
      material.uniforms.uTime.value = acceleratedTime;
      bhMaterial.uniforms.uTime.value = acceleratedTime;
      
      // Slightly faster rotation for more dynamic effect
      points.rotation.z += 0.0005; // Increased from 0.0003
      blackHole.rotation.z += 0.0008; // Increased from 0.0005
      
      renderer.render(scene, camera);
      animationRef.current = requestAnimationFrame(animate);
    };
    animate();

    /* ------------------------------------------------------------
     * Cleanup
     * ---------------------------------------------------------- */
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
      window.removeEventListener('resize', onResize);
      renderer.dispose();
      geometry.dispose();
      bhGeometry.dispose();
      material.dispose();
      bhMaterial.dispose();
      if (mountRef.current && renderer.domElement.parentNode) {
        mountRef.current.removeChild(renderer.domElement);
      }
    };
  }, []);

  return (
    <div 
      ref={mountRef} 
      className={`fixed inset-0 pointer-events-none z-0 ${className}`}
      style={{ mixBlendMode: 'screen' }}
    />
  );
};

export default SupernovaAnimation; 
