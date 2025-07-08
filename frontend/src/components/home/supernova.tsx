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

    // Particle system setup - optimized count for performance
    const particleCount = 50000;
    const positions = new Float32Array(particleCount * 3);
    const velocities = new Float32Array(particleCount * 3);
    const ages = new Float32Array(particleCount);
    const colors = new Float32Array(particleCount * 3);

    // Initialize particles
    for (let i = 0; i < particleCount; i++) {
      const i3 = i * 3;
      
      // Start particles clustered at center
      positions[i3] = (Math.random() - 0.5) * 0.1;
      positions[i3 + 1] = (Math.random() - 0.5) * 0.1;
      positions[i3 + 2] = (Math.random() - 0.5) * 0.1;
      
      // Random velocities for explosion
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.random() * Math.PI;
      const speed = Math.random() * 2 + 0.5;
      
      velocities[i3] = Math.sin(phi) * Math.cos(theta) * speed;
      velocities[i3 + 1] = Math.sin(phi) * Math.sin(theta) * speed;
      velocities[i3 + 2] = Math.cos(phi) * speed;
      
      ages[i] = Math.random() * 100;
      
      // Color based on temperature/age
      const heat = Math.random();
      colors[i3] = 1.0; // Red
      colors[i3 + 1] = heat * 0.8; // Green
      colors[i3 + 2] = heat * 0.3; // Blue
    }

    // Vertex shader for particle animation
    const vertexShader = `
      attribute float age;
      attribute vec3 velocity;
      attribute vec3 color;
      
      uniform float time;
      uniform float phase; // 0 = implosion, 1 = explosion, 2 = expansion
      uniform float explosionTime;
      
      varying vec3 vColor;
      varying float vAlpha;
      varying float vSize;
      varying vec3 vPosition;
      
      // Noise function for organic motion
      float noise(vec3 p) {
        return fract(sin(dot(p, vec3(12.9898, 78.233, 54.321))) * 43758.5453);
      }
      
      vec3 hsv2rgb(vec3 c) {
        vec4 K = vec4(1.0, 2.0 / 3.0, 1.0 / 3.0, 3.0);
        vec3 p = abs(fract(c.xxx + K.xyz) * 6.0 - K.www);
        return c.z * mix(K.xxx, clamp(p - K.xxx, 0.0, 1.0), c.y);
      }
      
      void main() {
        vec3 pos = position;
        float particleAge = age + time * 0.5;
        float lifespan = 1.0;
        
        // Phase-based animation with enhanced effects
        if (phase < 1.0) {
          // Implosion phase - dramatic collapse
          float implosionForce = pow(1.0 - phase, 3.0);
          pos *= implosionForce * 0.05;
          vSize = 1.0;
        } else if (phase < 2.0) {
          // Explosion phase - violent expansion
          float explosionProgress = smoothstep(0.0, 1.0, phase - 1.0);
          pos += velocity * explosionProgress * 4.0;
          
          // Add some chaotic motion during explosion
          float chaos = noise(pos + time) * 0.3;
          pos += velocity * chaos * explosionProgress;
          
          vSize = 3.0 + explosionProgress * 4.0;
        } else {
          // Expansion phase - beautiful slow growth with plasma-like motion
          float expansionTime = (phase - 2.0) * 0.3; // Slower expansion
          pos += velocity * (4.0 + expansionTime * 2.0);
          
          // Complex wave patterns for plasma-like motion
          float dist = length(pos);
          float wave1 = sin(particleAge * 0.2 + dist * 0.8) * 0.15;
          float wave2 = cos(particleAge * 0.15 + dist * 0.6) * 0.1;
          float spiral = sin(atan(pos.y, pos.x) * 8.0 + time * 0.5) * 0.05;
          
          vec3 waveOffset = normalize(pos) * (wave1 + wave2) + vec3(0.0, 0.0, spiral);
          pos += waveOffset;
          
          vSize = 2.0 + sin(particleAge * 0.3) * 0.5;
        }
        
        // Enhanced color based on phase and position
        vec3 finalColor = color;
        float temp = length(velocity) / 3.0;
        
        if (phase >= 2.0) {
          // Expansion phase: shift to cooler, more varied colors
          float hue = mod(temp + time * 0.1 + length(pos) * 0.05, 1.0);
          float saturation = 0.8 + sin(particleAge * 0.2) * 0.2;
          float brightness = 0.7 + temp * 0.3;
          finalColor = hsv2rgb(vec3(hue, saturation, brightness));
        } else if (phase >= 1.0) {
          // Explosion phase: hot white/yellow/orange
          finalColor = mix(vec3(1.0, 0.8, 0.2), vec3(1.0, 1.0, 1.0), temp);
        }
        
        // Calculate alpha with distance fade and phase effects
        float alpha = 1.0;
        if (phase >= 2.0) {
          float dist = length(pos);
          alpha = max(0.05, 1.0 - dist * 0.03);
          alpha *= 0.6 + sin(particleAge * 0.4) * 0.4; // Pulsing effect
        } else if (phase >= 1.0) {
          alpha = 0.8 + (phase - 1.0) * 0.2;
        }
        
        vColor = finalColor;
        vAlpha = alpha;
        vPosition = pos;
        
        vec4 mvPosition = modelViewMatrix * vec4(pos, 1.0);
        gl_Position = projectionMatrix * mvPosition;
        
        // Dynamic point size
        gl_PointSize = vSize * (200.0 / -mvPosition.z);
      }
    `;

    // Fragment shader for particle rendering
    const fragmentShader = `
      varying vec3 vColor;
      varying float vAlpha;
      varying float vSize;
      varying vec3 vPosition;
      
      uniform float time;
      uniform float phase;
      
      void main() {
        // Create circular particles with enhanced effects
        vec2 cxy = 2.0 * gl_PointCoord - 1.0;
        float dist = dot(cxy, cxy);
        
        if (dist > 1.0) {
          discard;
        }
        
        // Enhanced particle rendering based on phase
        float alpha = vAlpha;
        vec3 color = vColor;
        
        if (phase >= 2.0) {
          // Expansion phase: create plasma-like particles with cores
          float core = 1.0 - smoothstep(0.0, 0.3, dist);
          float halo = 1.0 - smoothstep(0.3, 1.0, dist);
          
          // Multi-layered color effect
          vec3 coreColor = color * 2.0;
          vec3 haloColor = color * 0.7;
          
          // Mix core and halo
          color = mix(haloColor, coreColor, core);
          alpha *= (core + halo * 0.5);
          
          // Add energy fluctuations
          float energy = sin(time * 3.0 + length(vPosition) * 0.5) * 0.2 + 0.8;
          color *= energy;
          
        } else if (phase >= 1.0) {
          // Explosion phase: bright, intense particles
          float intensity = 1.0 - smoothstep(0.0, 0.8, dist);
          color *= 1.5 + intensity;
          alpha *= intensity * 0.9;
          
        } else {
          // Implosion phase: dense, focused particles
          float focus = 1.0 - smoothstep(0.0, 0.5, dist);
          alpha *= focus;
        }
        
        // Soft edges for all phases
        alpha *= (1.0 - dist * 0.3);
        
        gl_FragColor = vec4(color, alpha);
      }
    `;

    // Create geometry and material
    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geometry.setAttribute('velocity', new THREE.BufferAttribute(velocities, 3));
    geometry.setAttribute('age', new THREE.BufferAttribute(ages, 1));
    geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));

    const material = new THREE.ShaderMaterial({
      vertexShader,
      fragmentShader,
      uniforms: {
        time: { value: 0 },
        phase: { value: 0 },
        explosionTime: { value: 0 }
      },
      blending: THREE.AdditiveBlending,
      depthTest: false,
      transparent: true,
    });

    const particles = new THREE.Points(geometry, material);
    scene.add(particles);

    camera.position.z = 5;

    // Animation variables
    let startTime = Date.now();
    let phase = 0; // 0 = implosion, 1 = explosion, 2 = expansion
    let phaseStartTime = startTime;

    // Animation loop
    const animate = () => {
      const currentTime = Date.now();
      const elapsed = (currentTime - startTime) / 1000;
      const phaseElapsed = (currentTime - phaseStartTime) / 1000;

      // Phase transitions - optimized timing for dramatic effect
      if (phase === 0 && phaseElapsed > 0.6) {
        // Quick transition from implosion to explosion
        phase = 1;
        phaseStartTime = currentTime;
      } else if (phase === 1 && phaseElapsed > 0.4) {
        // Transition from explosion to slow expansion
        phase = 2;
        phaseStartTime = currentTime;
      }

      // Update uniforms with adjusted timing
      material.uniforms.time.value = elapsed;
      
      if (phase === 0) {
        material.uniforms.phase.value = Math.min(1, phaseElapsed / 0.6);
      } else if (phase === 1) {
        material.uniforms.phase.value = 1 + Math.min(1, phaseElapsed / 0.4);
      } else {
        material.uniforms.phase.value = 2 + phaseElapsed * 0.05; // Very slow expansion for beautiful effect
      }

      // Rotate the particle system slowly
      particles.rotation.y += 0.002;
      particles.rotation.x += 0.001;

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
      geometry.dispose();
      material.dispose();
    };
  }, []);

  return (
    <div 
      ref={mountRef} 
      className={`fixed inset-0 pointer-events-none z-0 ${className}`}
      style={{
        background: 'radial-gradient(ellipse at center, rgba(5,5,15,0) 0%, rgba(5,5,15,0.3) 40%, rgba(0,0,5,0.6) 100%)',
        mixBlendMode: 'normal'
      }}
    />
  );
};

export default SupernovaAnimation; 
