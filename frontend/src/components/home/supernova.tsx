import React from 'react';

export const SpiralGalaxyAnimation: React.FC<{ zIndex?: number }> = ({ zIndex = -1 }) => {
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
      {/* Blank component */}
    </div>
  );
};

export default SpiralGalaxyAnimation; 
