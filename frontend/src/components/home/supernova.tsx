import React, { useRef, useEffect } from 'react';
import * as THREE from 'three';

interface SupernovaProps {
  className?: string;
}

/**
 * SupernovaAnimation
 *
 * A GPU particle shader that simulates a colourful supernova
 * burst which expands, slows, and eventually freezes to form a nebula. Ideal
 * for the hero section background, positioned slightly to the right so that
 * foreground text/content remains readable.
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
    camera.position.set(0, 0, 10);
    camera.lookAt(new THREE.Vector3(0, 0, 0));

    const renderer = new THREE.WebGLRenderer({ alpha: true, antialias: false });
    renderer.setPixelRatio(window.devicePixelRatio);
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setClearColor(0x000000, 0);
    mountRef.current.appendChild(renderer.domElement);

    /* ------------------------------------------------------------
     * Particle Buffer Setup
     * ---------------------------------------------------------- */
    const PARTICLE_COUNT = 70000;
    const initPositions = new Float32Array(PARTICLE_COUNT * 3);
    const directions    = new Float32Array(PARTICLE_COUNT * 3);
    const speeds        = new Float32Array(PARTICLE_COUNT);
    const startTimes    = new Float32Array(PARTICLE_COUNT);

    for (let i = 0; i < PARTICLE_COUNT; i++) {
      const i3 = i * 3;

      // Start at (almost) the center with tiny jitter to avoid z-fighting
      initPositions[i3]     = THREE.MathUtils.randFloatSpread(0.2);
      initPositions[i3 + 1] = THREE.MathUtils.randFloatSpread(0.2);
      initPositions[i3 + 2] = THREE.MathUtils.randFloatSpread(0.2);

      // Random direction over the sphere
      const theta = Math.random() * Math.PI * 2;
      const phi   = Math.acos(THREE.MathUtils.randFloatSpread(2)); // [-1,1]
      directions[i3]     = Math.sin(phi) * Math.cos(theta);
      directions[i3 + 1] = Math.sin(phi) * Math.sin(theta);
      directions[i3 + 2] = Math.cos(phi);

      speeds[i]     = THREE.MathUtils.randFloat(4, 10); // slower expansion
      startTimes[i] = Math.random() * 0.5; // slight stagger for natural burst
    }

    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute('aInitPos',   new THREE.BufferAttribute(initPositions, 3));
    geometry.setAttribute('aDirection', new THREE.BufferAttribute(directions, 3));
    geometry.setAttribute('aSpeed',     new THREE.BufferAttribute(speeds, 1));
    geometry.setAttribute('aStartTime', new THREE.BufferAttribute(startTimes, 1));
    // Provide a default position attribute for frustum culling & bounding calculations
    geometry.setAttribute('position',   new THREE.BufferAttribute(initPositions, 3));
    geometry.computeBoundingSphere();

    /* ------------------------------------------------------------
     * Shaders
     * ---------------------------------------------------------- */
    const vertexShader = /* glsl */`
      precision highp float;

      attribute vec3  aInitPos;
      attribute vec3  aDirection;
      attribute float aSpeed;
      attribute float aStartTime;

      uniform float uTime;
      uniform float uStartDelay;       // Global 500 ms explosion delay
      uniform float uFreezeStart;
      uniform float uFreezeDuration;
      uniform float uDrag;
      uniform float uEntropyAmp;
      uniform float uEntropyFreq;

      varying float vDist;

      void main() {
        float t = max(uTime - uStartDelay - aStartTime, 0.0);

        // Progress toward freeze (0 → 1)
        float freezeProgress = clamp((t - uFreezeStart) / uFreezeDuration, 0.0, 1.0);

        // Integrate decaying velocity analytically for smooth slowdown
        float distFactor = (1.0 - exp(-uDrag * t)) * (aSpeed / uDrag);

        vec3 pos = aInitPos + aDirection * distFactor;

        // Subtle swirl for nebula-like wisps
        vec3 swirlAxis = vec3(0.0, 0.0, 1.0);
        vec3 swirlDir  = normalize(cross(swirlAxis, aDirection));
        float swirlAmp = mix(0.35, 0.05, freezeProgress); // keep some swirl after freeze
        pos += swirlDir * swirlAmp * sin(t * 0.8);

        // Smooth entropy-based drift (slow wobble)
        float entropyPhase = uTime * uEntropyFreq;
        vec3 entropyOffset = vec3(
          sin(aInitPos.x * 3.1 + entropyPhase),
          cos(aInitPos.y * 2.7 + entropyPhase * 1.1),
          sin(aInitPos.z * 3.7 - entropyPhase * 0.9)
        ) * uEntropyAmp * freezeProgress;
        pos += entropyOffset;

        vDist = length(pos);

        // Project
        gl_Position = projectionMatrix * modelViewMatrix * vec4(pos, 1.0);

        // Keep particles larger to appear as smoke; minimal shrink
        float size = mix(6.0, 5.5, freezeProgress);
        gl_PointSize = size * (80.0 / - (modelViewMatrix * vec4(pos, 1.0)).z);
      }
    `;

    const fragmentShader = /* glsl */`
      precision highp float;

      varying float vDist;
      uniform float uTime;

      // Minimal HSV→RGB
      vec3 hsv2rgb(vec3 c) {
        vec3 rgb = clamp(abs(mod(c.x * 6.0 + vec3(0.0, 4.0, 2.0), 6.0) - 3.0) - 1.0, 0.0, 1.0);
        rgb = rgb * rgb * (3.0 - 2.0 * rgb);
        return c.z * mix(vec3(1.0), rgb, c.y);
      }

      void main() {
        // Circular fade for each point
        vec2 uv = gl_PointCoord * 2.0 - 1.0;
        float alphaCircle = pow(1.0 - dot(uv, uv), 1.5); // softer edges => smoke

        // Distance-based hue shift with gentle temporal scroll
        float hue = mod(0.65 - vDist * 0.08 + uTime * 0.015, 1.0);
        float sat = 0.6; // slightly less saturated for smoky feel
        float val = 0.9;
        vec3 color = hsv2rgb(vec3(hue, sat, val));

        gl_FragColor = vec4(color, alphaCircle * 0.85);
      }
    `;

    const material = new THREE.ShaderMaterial({
      uniforms: {
        uTime:           { value: 0 },
        uStartDelay:     { value: 0.5 },   // 500 ms global delay
        uFreezeStart:    { value: 6.0 },   // seconds until freeze begins
        uFreezeDuration: { value: 4.0 },   // seconds to fully freeze
        uDrag:           { value: 0.25 },
        uEntropyAmp:     { value: 0.25 },  // smoke drift amplitude
        uEntropyFreq:    { value: 0.2 }
      },
      vertexShader,
      fragmentShader,
      transparent: true,
      depthWrite: false,
      blending: THREE.NormalBlending  // smoother smoke-like accumulation
    });

    const points = new THREE.Points(geometry, material);
    points.position.x = 2.0; // shift right so main content has space
    scene.add(points);

    /* ------------------------------------------------------------
     * Animation Loop
     * ---------------------------------------------------------- */
    const start = performance.now();

    const onResize = () => {
      camera.aspect = window.innerWidth / window.innerHeight;
      camera.updateProjectionMatrix();
      renderer.setSize(window.innerWidth, window.innerHeight);
    };
    window.addEventListener('resize', onResize);

    const tick = () => {
      const elapsed = (performance.now() - start) / 1000;
      material.uniforms.uTime.value = elapsed;

      renderer.render(scene, camera);
      animationRef.current = requestAnimationFrame(tick);
    };
    tick();

    /* ------------------------------------------------------------
     * Cleanup
     * ---------------------------------------------------------- */
    return () => {
      cancelAnimationFrame(animationRef.current!);
      window.removeEventListener('resize', onResize);
      renderer.dispose();
      geometry.dispose();
      material.dispose();
      if (mountRef.current) {
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
