import React, { useEffect, useRef } from 'react';
import * as THREE from 'three';

interface SupernovaProps {
  className?: string;
}

export const SupernovaAnimation: React.FC<SupernovaProps> = ({ className = '' }) => {
  const mountRef = useRef<HTMLDivElement>(null);
  const sceneRef = useRef<THREE.Scene | null>(null);
  const rendererRef = useRef<THREE.WebGLRenderer | null>(null);
  const animationIdRef = useRef<number | null>(null);

  useEffect(() => {
    if (!mountRef.current) return;

    // Scene setup
    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 1000);
    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setClearColor(0x000000, 0);
    mountRef.current.appendChild(renderer.domElement);

    sceneRef.current = scene;
    rendererRef.current = renderer;

    // GPU Computation setup for gravity simulation
    const textureSize = 256; // 256x256 = 65536 particles
    const particleCount = textureSize * textureSize;

    // Create render targets for position and velocity
    const positionTarget1 = new THREE.WebGLRenderTarget(textureSize, textureSize, {
      minFilter: THREE.NearestFilter,
      magFilter: THREE.NearestFilter,
      format: THREE.RGBAFormat,
      type: THREE.FloatType
    });

    const positionTarget2 = new THREE.WebGLRenderTarget(textureSize, textureSize, {
      minFilter: THREE.NearestFilter,
      magFilter: THREE.NearestFilter,
      format: THREE.RGBAFormat,
      type: THREE.FloatType
    });

    const velocityTarget1 = new THREE.WebGLRenderTarget(textureSize, textureSize, {
      minFilter: THREE.NearestFilter,
      magFilter: THREE.NearestFilter,
      format: THREE.RGBAFormat,
      type: THREE.FloatType
    });

    const velocityTarget2 = new THREE.WebGLRenderTarget(textureSize, textureSize, {
      minFilter: THREE.NearestFilter,
      magFilter: THREE.NearestFilter,
      format: THREE.RGBAFormat,
      type: THREE.FloatType
    });

    // Initialize particle data
    const positions = new Float32Array(particleCount * 4);
    const velocities = new Float32Array(particleCount * 4);

    for (let i = 0; i < particleCount; i++) {
      const i4 = i * 4;
      
      // Random positions around two centers
      const center = Math.random() < 0.5 ? 0 : 1;
      const centerX = center === 0 ? -3 : 3;
      const centerY = center === 0 ? -1 : 1;
      
      const angle = Math.random() * Math.PI * 2;
      const radius = Math.random() * 2 + 0.5;
      
      positions[i4] = centerX + Math.cos(angle) * radius;
      positions[i4 + 1] = centerY + Math.sin(angle) * radius;
      positions[i4 + 2] = (Math.random() - 0.5) * 0.5;
      positions[i4 + 3] = Math.random(); // particle type/mass
      
      // Initial orbital velocities
      const orbitalSpeed = 0.3;
      velocities[i4] = -Math.sin(angle) * orbitalSpeed * (center === 0 ? 1 : -1);
      velocities[i4 + 1] = Math.cos(angle) * orbitalSpeed * (center === 0 ? 1 : -1);
      velocities[i4 + 2] = 0;
      velocities[i4 + 3] = 0; // unused
    }

    // Create initial textures
    const positionTexture = new THREE.DataTexture(positions, textureSize, textureSize, THREE.RGBAFormat, THREE.FloatType);
    positionTexture.needsUpdate = true;
    
    const velocityTexture = new THREE.DataTexture(velocities, textureSize, textureSize, THREE.RGBAFormat, THREE.FloatType);
    velocityTexture.needsUpdate = true;

         // Position update shader
     const positionShader = new THREE.ShaderMaterial({
       uniforms: {
         u_positionTexture: { value: positionTexture },
         u_velocityTexture: { value: velocityTexture },
         u_time: { value: 0 },
         u_delta: { value: 0.016 },
         resolution: { value: new THREE.Vector2(textureSize, textureSize) }
       },
       vertexShader: `
         void main() {
           gl_Position = vec4(position, 1.0);
         }
       `,
       fragmentShader: `
         uniform sampler2D u_positionTexture;
         uniform sampler2D u_velocityTexture;
         uniform float u_delta;
         uniform vec2 resolution;
         
         void main() {
           vec2 uv = gl_FragCoord.xy / resolution.xy;
           vec3 pos = texture2D(u_positionTexture, uv).xyz;
           vec3 vel = texture2D(u_velocityTexture, uv).xyz;
           
           // Update position using velocity
           pos += vel * u_delta;
           
           gl_FragColor = vec4(pos, 1.0);
         }
       `
     });

         // Velocity update shader (gravity)
     const velocityShader = new THREE.ShaderMaterial({
       uniforms: {
         u_positionTexture: { value: positionTexture },
         u_velocityTexture: { value: velocityTexture },
         u_time: { value: 0 },
         u_delta: { value: 0.016 },
         u_mass1Pos: { value: new THREE.Vector3(-3, -1, 0) },
         u_mass2Pos: { value: new THREE.Vector3(3, 1, 0) },
         u_phase: { value: 0 }, // 0 = approach, 1 = merge, 2 = explosion
         resolution: { value: new THREE.Vector2(textureSize, textureSize) }
       },
       vertexShader: `
         void main() {
           gl_Position = vec4(position, 1.0);
         }
       `,
       fragmentShader: `
         uniform sampler2D u_positionTexture;
         uniform sampler2D u_velocityTexture;
         uniform float u_time;
         uniform float u_delta;
         uniform vec3 u_mass1Pos;
         uniform vec3 u_mass2Pos;
         uniform float u_phase;
         uniform vec2 resolution;
         
         void main() {
           vec2 uv = gl_FragCoord.xy / resolution.xy;
          vec3 pos = texture2D(u_positionTexture, uv).xyz;
          vec3 vel = texture2D(u_velocityTexture, uv).xyz;
          
          vec3 force = vec3(0.0);
          
                     if (u_phase < 1.0) {
             // Approach phase - gravity from two masses
             vec3 r1 = u_mass1Pos - pos;
             vec3 r2 = u_mass2Pos - pos;
             
             float d1 = length(r1);
             float d2 = length(r2);
             
             // Stronger gravitational force
             float G = 8.0; // Increased from 2.0
             force += G / (d1 * d1 + 0.01) * normalize(r1);
             force += G / (d2 * d2 + 0.01) * normalize(r2);
             
           } else if (u_phase < 2.0) {
             // Merge phase - strong implosion toward center
             vec3 center = vec3(0.0, 0.0, 0.0); // Always center for merge
             vec3 r = center - pos;
             float d = length(r);
             
             // Very strong inward force during merge
             float mergeProgress = u_phase - 1.0; // 0 to 1
             float implosionForce = 25.0 * mergeProgress; // Much stronger
             force += implosionForce / (d + 0.01) * normalize(r);
             
           } else {
             // Explosion phase - strong outward force from center
             vec3 center = vec3(0.0, 0.0, 0.0);
             vec3 r = pos - center;
             float d = length(r);
             
             // Strong explosive outward force
             float explosionProgress = u_phase - 2.0;
             float explosionForce = 15.0 * (1.0 + explosionProgress * 2.0);
             
             if (d > 0.01) {
               force += explosionForce * normalize(r);
             } else {
               // For particles at center, give random outward velocity
               force += vec3(
                 sin(u_time * 10.0 + pos.x * 100.0) * 10.0,
                 cos(u_time * 10.0 + pos.y * 100.0) * 10.0,
                 sin(u_time * 15.0 + pos.z * 100.0) * 5.0
               );
             }
             
             // Add turbulence for realistic explosion
             float noise1 = sin(pos.x * 8.0 + u_time * 2.0) * cos(pos.y * 6.0 + u_time);
             float noise2 = cos(pos.y * 7.0 + u_time * 1.5) * sin(pos.z * 5.0);
             force += vec3(noise1, noise2, noise1 * 0.5) * 2.0;
           }
          
                     // Update velocity
           vel += force * u_delta;
           
           // Phase-based damping
           float damping = u_phase < 2.0 ? 0.998 : 0.9995; // Less damping during explosion
           vel *= damping;
          
          gl_FragColor = vec4(vel, 1.0);
        }
      `
    });

    // Create scene for GPU computation
    const computeScene = new THREE.Scene();
    const computeCamera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1);
    const quad = new THREE.Mesh(new THREE.PlaneGeometry(2, 2), positionShader);
    computeScene.add(quad);

    // Particle rendering setup
    const particleGeometry = new THREE.BufferGeometry();
    const indices = new Float32Array(particleCount);
    const renderPositions = new Float32Array(particleCount * 3);

    for (let i = 0; i < particleCount; i++) {
      indices[i] = i;
      const x = (i % textureSize) / textureSize;
      const y = Math.floor(i / textureSize) / textureSize;
      renderPositions[i * 3] = x;
      renderPositions[i * 3 + 1] = y;
      renderPositions[i * 3 + 2] = 0;
    }

         particleGeometry.setAttribute('position', new THREE.BufferAttribute(renderPositions, 3));
     particleGeometry.setAttribute('a_texCoord', new THREE.BufferAttribute(renderPositions, 3));

    // Particle rendering shader
    const particleMaterial = new THREE.ShaderMaterial({
      uniforms: {
        u_positionTexture: { value: null },
        u_time: { value: 0 },
        u_phase: { value: 0 },
        u_textureSize: { value: textureSize }
      },
             vertexShader: `
         uniform sampler2D u_positionTexture;
         uniform float u_time;
         uniform float u_phase;
         uniform float u_textureSize;
         
         attribute vec3 a_texCoord;
         varying vec3 vColor;
         varying float vAlpha;
         
         vec3 hsv2rgb(vec3 c) {
           vec4 K = vec4(1.0, 2.0 / 3.0, 1.0 / 3.0, 3.0);
           vec3 p = abs(fract(c.xxx + K.xyz) * 6.0 - K.www);
           return c.z * mix(K.xxx, clamp(p - K.xxx, 0.0, 1.0), c.y);
         }
         
         void main() {
           vec2 texCoord = a_texCoord.xy;
          vec4 posData = texture2D(u_positionTexture, texCoord);
          vec3 pos = posData.xyz;
          
          // Color based on phase and position
          vec3 color = vec3(0.8, 0.4, 0.2); // Default warm color
          float alpha = 0.6;
          
                     if (u_phase < 1.0) {
             // Approach phase - blue/white orbital material
             float orbitGlow = sin(u_time * 2.0 + length(pos) * 3.0) * 0.3 + 0.7;
             color = vec3(0.2, 0.4, 0.9) * orbitGlow;
             alpha = 0.5;
           } else if (u_phase < 2.0) {
             // Merge phase - bright white/yellow core
             float mergeIntensity = (u_phase - 1.0) * 3.0 + 1.0;
             color = vec3(1.0, 0.9, 0.7) * mergeIntensity;
             alpha = 0.9;
           } else {
             // Explosion phase - dramatic colorful expanding shockwave
             float dist = length(pos);
             float speed = length(texture2D(u_positionTexture, texCoord).xyz - pos);
             
             // Color based on distance and speed
             float hue = mod(dist * 0.2 + speed * 0.5 + u_time * 0.05, 1.0);
             float saturation = 0.8 + sin(u_time * 3.0 + dist * 2.0) * 0.2;
             float brightness = 0.6 + sin(u_time * 2.0 + dist) * 0.3;
             
             color = hsv2rgb(vec3(hue, saturation, brightness));
             alpha = max(0.05, 0.7 - dist * 0.08);
             
             // Extra glow for recently exploded particles
             if (u_phase < 3.0) {
               float recentExplosion = 1.0 - (u_phase - 2.0);
               color *= 1.0 + recentExplosion * 2.0;
               alpha *= 1.0 + recentExplosion;
             }
           }
          
          vColor = color;
          vAlpha = alpha;
          
          vec4 mvPosition = modelViewMatrix * vec4(pos, 1.0);
          gl_Position = projectionMatrix * mvPosition;
          
                     // Dynamic particle size based on phase
           float size;
           if (u_phase < 1.0) {
             size = 1.5; // Small during approach
           } else if (u_phase < 2.0) {
             size = 6.0 + (u_phase - 1.0) * 8.0; // Growing during merge
           } else {
             float dist = length(pos);
             size = 3.0 + sin(u_time * 4.0 + dist * 2.0) * 1.0; // Pulsing during explosion
           }
           
           gl_PointSize = size * (80.0 / -mvPosition.z);
        }
      `,
      fragmentShader: `
        varying vec3 vColor;
        varying float vAlpha;
        
        void main() {
          vec2 cxy = 2.0 * gl_PointCoord - 1.0;
          float dist = dot(cxy, cxy);
          
          if (dist > 1.0) discard;
          
          float alpha = vAlpha * (1.0 - dist) * 0.8;
          gl_FragColor = vec4(vColor, alpha);
        }
      `,
      blending: THREE.AdditiveBlending,
      depthTest: false,
      transparent: true
    });

    const particles = new THREE.Points(particleGeometry, particleMaterial);
    scene.add(particles);

    // Position camera to show supernova center-right
    camera.position.set(-2, 0, 8);
    camera.lookAt(0.5, 0, 0);

    // Animation variables
    let startTime = Date.now();
    let phase = 0; // 0 = approach, 1 = merge, 2 = explosion
    let phaseStartTime = startTime;
    let currentPositionTarget = positionTarget1;
    let currentVelocityTarget = velocityTarget1;

    // Animation loop
    const animate = () => {
      const currentTime = Date.now();
      const elapsed = (currentTime - startTime) / 1000;
      const phaseElapsed = (currentTime - phaseStartTime) / 1000;

             // Phase transitions with debugging
       if (phase === 0 && phaseElapsed > 3.0) {
         // Transition to merge phase
         console.log('Phase 0 -> 1: Merging phase started');
         phase = 1;
         phaseStartTime = currentTime;
       } else if (phase === 1 && phaseElapsed > 0.8) {
         // Transition to explosion phase
         console.log('Phase 1 -> 2: Explosion phase started');
         phase = 2;
         phaseStartTime = currentTime;
       }

             // Update mass positions (they move toward each other in approach phase)
       let mass1Pos = new THREE.Vector3(-3, -1, 0);
       let mass2Pos = new THREE.Vector3(3, 1, 0);
       
       if (phase === 0) {
         const t = Math.min(1, phaseElapsed / 3.0);
         mass1Pos.lerp(new THREE.Vector3(-0.3, 0, 0), t);
         mass2Pos.lerp(new THREE.Vector3(0.3, 0, 0), t);
       } else {
         // During merge and explosion, masses are at center
         mass1Pos.set(0, 0, 0);
         mass2Pos.set(0, 0, 0);
       }

       // Update compute shader uniforms
       const phaseValue = phase === 0 ? phaseElapsed / 3.0 : 
                         phase === 1 ? 1 + phaseElapsed / 0.8 : 
                         2 + (phaseElapsed * 0.3);

             positionShader.uniforms.u_time.value = elapsed;
       velocityShader.uniforms.u_time.value = elapsed;
       velocityShader.uniforms.u_mass1Pos.value = mass1Pos;
       velocityShader.uniforms.u_mass2Pos.value = mass2Pos;
       velocityShader.uniforms.u_phase.value = phaseValue;

       // Debug logging every few seconds
       if (Math.floor(elapsed) % 2 === 0 && elapsed - Math.floor(elapsed) < 0.1) {
         console.log(`Phase: ${phase}, PhaseValue: ${phaseValue.toFixed(2)}, Elapsed: ${phaseElapsed.toFixed(1)}s`);
       }

      // GPU computation step
      // Update velocities
      quad.material = velocityShader;
      velocityShader.uniforms.u_positionTexture.value = currentPositionTarget.texture;
      velocityShader.uniforms.u_velocityTexture.value = currentVelocityTarget.texture;
      
      const nextVelocityTarget = currentVelocityTarget === velocityTarget1 ? velocityTarget2 : velocityTarget1;
      renderer.setRenderTarget(nextVelocityTarget);
      renderer.render(computeScene, computeCamera);

      // Update positions
      quad.material = positionShader;
      positionShader.uniforms.u_positionTexture.value = currentPositionTarget.texture;
      positionShader.uniforms.u_velocityTexture.value = nextVelocityTarget.texture;
      
      const nextPositionTarget = currentPositionTarget === positionTarget1 ? positionTarget2 : positionTarget1;
      renderer.setRenderTarget(nextPositionTarget);
      renderer.render(computeScene, computeCamera);

      // Swap targets
      currentPositionTarget = nextPositionTarget;
      currentVelocityTarget = nextVelocityTarget;

      // Update particle rendering
      particleMaterial.uniforms.u_positionTexture.value = currentPositionTarget.texture;
      particleMaterial.uniforms.u_time.value = elapsed;
      particleMaterial.uniforms.u_phase.value = phaseValue;

      // Render to screen
      renderer.setRenderTarget(null);
      renderer.render(scene, camera);

      animationIdRef.current = requestAnimationFrame(animate);
    };

    animate();

    // Handle resize
    const handleResize = () => {
      camera.aspect = window.innerWidth / window.innerHeight;
      camera.updateProjectionMatrix();
      renderer.setSize(window.innerWidth, window.innerHeight);
    };

    window.addEventListener('resize', handleResize);

    // Cleanup
    return () => {
      if (animationIdRef.current) {
        cancelAnimationFrame(animationIdRef.current);
      }
      window.removeEventListener('resize', handleResize);
      if (mountRef.current && renderer.domElement) {
        mountRef.current.removeChild(renderer.domElement);
      }
      renderer.dispose();
      particleGeometry.dispose();
      particleMaterial.dispose();
      positionTarget1.dispose();
      positionTarget2.dispose();
      velocityTarget1.dispose();
      velocityTarget2.dispose();
    };
  }, []);

  return (
    <div 
      ref={mountRef} 
      className={`fixed inset-0 pointer-events-none z-0 ${className}`}
      style={{
        background: 'radial-gradient(ellipse at 65% center, rgba(5,5,15,0) 0%, rgba(5,5,15,0.1) 40%, rgba(0,0,5,0.2) 100%)',
        mixBlendMode: 'normal'
      }}
    />
  );
};

export default SupernovaAnimation; 
