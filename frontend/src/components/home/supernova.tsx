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
    let renderer: any, scene: any, camera: any, quasar: any;
    let width = window.innerWidth;
    let height = window.innerHeight;
    let frameId: number;
    let THREE: any;

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
      camera.position.z = 150;

      // --- Quasar (center of gravity) ---
      // Placeholder: bright sphere, ready for shader
      const quasarGeometry = new THREE.SphereGeometry(2, 32, 32);
      const quasarMaterial = new THREE.MeshBasicMaterial({ color: 0xffffcc });
      quasar = new THREE.Mesh(quasarGeometry, quasarMaterial);
      scene.add(quasar);

      // --- Particles (explosion + orbit) ---
      const particleCount = 300;
      const particleGeometry = new THREE.BufferGeometry();
      const positions: number[] = [];
      // Store per-particle state: [radius, angle, y, radialSpeed, angularSpeed]
      const particles: { r: number; theta: number; y: number; radialSpeed: number; angularSpeed: number; orbiting: boolean; phi: number; }[] = [];
      const ORBIT_RADIUS = 120;
      const EPSILON = 0.01; // how close to orbit before clamping
      for (let i = 0; i < particleCount; i++) {
        // Spherical distribution
        const theta = Math.random() * 2 * Math.PI; // azimuthal
        const phi = Math.acos(2 * Math.random() - 1); // polar, uniform sphere
        const r = Math.random() * 2 + 2;
        // Spherical to Cartesian
        const x = Math.sin(phi) * Math.cos(theta) * r;
        const y = Math.cos(phi) * r;
        const z = Math.sin(phi) * Math.sin(theta) * r;
        // Outward radial speed
        const radialSpeed = 2.5 + Math.random() * 2.5;
        // Slow orbit: random direction, but always tangent to sphere
        // We'll use theta for azimuthal orbit, but keep phi fixed for simplicity
        const angularSpeed = (Math.random() - 0.5) * 0.0004;
        particles.push({
          r,
          theta,
          y,
          radialSpeed,
          angularSpeed,
          orbiting: false,
          phi // store phi for orbiting
        });
        positions.push(x, y, z);
      }
      particleGeometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
      // Simple white points, ready for shader
      const particleMaterial = new THREE.PointsMaterial({ color: 0xffffff, size: 0.3 });
      const particleSystem = new THREE.Points(particleGeometry, particleMaterial);
      scene.add(particleSystem);

      // --- Animation loop ---
      function animate() {
        const pos = particleGeometry.attributes.position.array as number[];
        for (let i = 0; i < particleCount; i++) {
          let p = particles[i];
          // Always update angle for spiral motion (now in 3D)
          p.theta += p.angularSpeed;
          // Ease out radius as it approaches the orbit radius
          if (!p.orbiting) {
            const distToOrbit = ORBIT_RADIUS - p.r;
            if (distToOrbit > EPSILON) {
              const ease = Math.max(0.01, distToOrbit / ORBIT_RADIUS);
              p.r += p.radialSpeed * ease * 0.5;
            } else {
              p.r = ORBIT_RADIUS;
              p.orbiting = true;
            }
          }
          // Spherical to Cartesian for position
          const x = Math.sin(p.phi) * Math.cos(p.theta) * p.r;
          const y = Math.cos(p.phi) * p.r;
          const z = Math.sin(p.phi) * Math.sin(p.theta) * p.r;
          pos[i * 3] = x;
          pos[i * 3 + 1] = y;
          pos[i * 3 + 2] = z;
        }
        particleGeometry.attributes.position.needsUpdate = true;
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
        renderer.dispose();
        particleGeometry.dispose();
        particleMaterial.dispose();
        quasarGeometry.dispose();
        quasarMaterial.dispose();
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
