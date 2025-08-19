import { useRef, useEffect } from 'react'
import { gsap } from 'gsap'
import { TextPlugin } from 'gsap/TextPlugin'

// Register GSAP plugins
gsap.registerPlugin(TextPlugin)

// Advanced text reveal animation with sophisticated effects
function AdvancedAnimatedText({ 
  text, 
  className = "",
  delay = 0,
  animationType = "reveal"
}: { 
  text: string
  className?: string
  delay?: number
  animationType?: "reveal" | "typewriter" | "glitch" | "morphing"
}) {
  const textRef = useRef<HTMLSpanElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const element = textRef.current
    const container = containerRef.current
    if (!element || !container) return

    // Check for reduced motion preference
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    
    if (prefersReducedMotion) {
      gsap.set(element, { opacity: 1 })
      return
    }

    // Create master timeline for orchestrated animations
    const tl = gsap.timeline({ delay })

    switch (animationType) {
      case "reveal":
        // Advanced reveal with split text and staggered animation
        // Use Array.from to properly handle Unicode characters including emojis
        const chars = Array.from(text).map((char) => {
          const span = document.createElement('span')
          span.textContent = char === ' ' ? '\u00A0' : char
          span.style.display = 'inline-block'
          span.style.willChange = 'transform, opacity'
          return span
        })
        
        element.innerHTML = ''
        chars.forEach(char => element.appendChild(char))

        // Set initial state with rotation and advanced transforms
        gsap.set(chars, {
          opacity: 0,
          y: 20,
          rotationZ: () => gsap.utils.random(-45, 45), // Random initial rotation
          rotationX: -90,
          transformOrigin: "50% 50% -50px",
          filter: "blur(10px)",
          scale: 0.3
        })

        // Single unified animation - bounce and wave that settles to baseline
        tl.to(chars, {
          opacity: 1,
          y: 0, // Settle to baseline, no wave positions
          rotationZ: 0, // Correct rotation
          rotationX: 0,
          filter: "blur(0px)",
          scale: 1, // Final scale
          duration: 1.2, // Longer duration to include bounce and wave motion
          ease: "elastic.out(1, 0.5)", // Elastic ease creates bounce and wave in single animation
          stagger: {
            amount: 0.5,
            from: "start"
          }
        })
        .to(chars, {
          willChange: "auto"
        }, "-=0.2")

        // Add interactive hover effects for each character (no rotation)
        chars.forEach((char) => {
          char.addEventListener('mouseenter', () => {
            gsap.to(char, {
              scale: 1.2,
              y: -5,
              duration: 0.3,
              ease: "back.out(1.7)"
            })
          })
          
          char.addEventListener('mouseleave', () => {
            gsap.to(char, {
              scale: 1,
              y: 0, // Return to baseline
              duration: 0.3,
              ease: "power2.out"
            })
          })
        })
        break

      case "typewriter":
        // Sophisticated typewriter with cursor and realistic timing
        const cursor = document.createElement('span')
        cursor.textContent = '|'
        cursor.style.animation = 'blink 1s infinite'
        
        element.innerHTML = ''
        element.appendChild(cursor)

        tl.to({}, { duration: 0.5 }) // Initial pause
          .to(element, {
            text: {
              value: text,
              delimiter: ""
            },
            duration: text.length * 0.01, // typewriter speed
            ease: "none",
            onUpdate: () => {
              // Add natural typing variations
              if (Math.random() < 0.1) {
                tl.pause()
                setTimeout(() => tl.resume(), Math.random() * 100 + 50)
              }
            }
          })
          .to(cursor, {
            opacity: 0,
            duration: 0.3,
            delay: 1
          })
        break

      case "glitch":
        // Glitch effect with data corruption aesthetic
        const originalText = text
        element.textContent = originalText

        gsap.set(element, {
          filter: "blur(0px)",
          textShadow: "none"
        })

        tl.to(element, {
          duration: 0.1,
          repeat: 8,
          yoyo: true,
          ease: "power2.inOut",
          textShadow: "2px 0 #ff0000, -2px 0 #00ff00",
          filter: "blur(1px)",
          onUpdate: function() {
            if (Math.random() < 0.5) {
              const glitchChars = "!@#$%^&*(){}[]|\\:;<>?/~`"
              let glitchText = ""
              for (let i = 0; i < originalText.length; i++) {
                glitchText += Math.random() < 0.1 
                  ? glitchChars[Math.floor(Math.random() * glitchChars.length)]
                  : originalText[i]
              }
              element.textContent = glitchText
            }
          }
        })
        .to(element, {
          textShadow: "none",
          filter: "blur(0px)",
          duration: 0.2,
          onComplete: () => {
            element.textContent = originalText
          }
        })
        break

      case "morphing":
        // Text morphing with intermediate states
        const morphStates = [
          "...................",
          "▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓",
          "███████████████████",
          text
        ]

        element.textContent = morphStates[0]
        
        morphStates.forEach((state, index) => {
          if (index === 0) return
          tl.to(element, {
            text: { value: state },
            duration: 0.3,
            ease: "power2.inOut"
          }, index * 0.2)
        })
        break
    }

    // Cleanup function
    return () => {
      tl.kill()
      if (element) {
        gsap.set(element, { clearProps: "all" })
        // Clean up event listeners for reveal animation
        const charElements = element.querySelectorAll('span')
        charElements.forEach(char => {
          const newChar = char.cloneNode(true)
          char.parentNode?.replaceChild(newChar, char)
        })
      }
    }
  }, [text, delay, animationType])

  return (
    <div ref={containerRef} className="inline-block">
      <span ref={textRef} className={className} />
    </div>
  )
}

// Floating particle system for background enhancement
function FloatingParticles() {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    // Check for reduced motion and performance preferences
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    const isLowPerformance = navigator.hardwareConcurrency < 4 || window.devicePixelRatio < 2
    
    if (prefersReducedMotion) return

    // Adjust particle count based on performance
    const particleCount = isLowPerformance ? 8 : 20

    // Create particles with optimized DOM manipulation
    const fragment = document.createDocumentFragment()
    const particles: HTMLDivElement[] = []
    
    for (let i = 0; i < particleCount; i++) {
      const particle = document.createElement('div')
      particle.className = 'absolute w-1 h-1 bg-blue-500/20 rounded-full'
      particle.style.willChange = 'transform'
      particle.setAttribute('aria-hidden', 'true') // Hide from screen readers
      fragment.appendChild(particle)
      particles.push(particle)
    }
    
    container.appendChild(fragment)

    // Animate particles with performance optimizations
    particles.forEach((particle, i) => {
      gsap.set(particle, {
        x: Math.random() * window.innerWidth,
        y: Math.random() * window.innerHeight,
        scale: Math.random() * 0.5 + 0.5,
        force3D: true // Force hardware acceleration
      })

      gsap.to(particle, {
        x: `+=${Math.random() * 200 - 100}`,
        y: `+=${Math.random() * 200 - 100}`,
        rotation: 360,
        duration: Math.random() * 10 + 10,
        repeat: -1,
        yoyo: true,
        ease: "sine.inOut",
        delay: i * 0.1,
        force3D: true
      })
    })

    return () => {
      particles.forEach(particle => {
        gsap.killTweensOf(particle)
        particle.remove()
      })
    }
  }, [])

  return <div ref={containerRef} className="fixed inset-0 pointer-events-none z-0" />
}

// Sequential animation component that applies typewriter then glitch
function SequentialAnimatedText({ 
  text, 
  className = "",
  delay = 0
}: { 
  text: string
  className?: string
  delay?: number
}) {
  const textRef = useRef<HTMLSpanElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const element = textRef.current
    const container = containerRef.current
    if (!element || !container) return

    // Check for reduced motion preference
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    
    if (prefersReducedMotion) {
      gsap.set(element, { opacity: 1 })
      element.textContent = text
      return
    }

    // Create master timeline for sequential animations
    const tl = gsap.timeline({ delay })

    // Phase 1: Typewriter animation
    const cursor = document.createElement('span')
    cursor.textContent = '|'
    cursor.style.animation = 'blink 1s infinite'
    
    element.innerHTML = ''
    element.appendChild(cursor)

    tl.to({}, { duration: 0.5 }) // Initial pause
      .to(element, {
        text: {
          value: text,
          delimiter: ""
        },
        duration: text.length * 0.03, // Same speed as before
        ease: "none",
        onUpdate: () => {
          // Add natural typing variations
          if (Math.random() < 0.1) {
            tl.pause()
            setTimeout(() => tl.resume(), Math.random() * 100 + 50)
          }
        }
      })
      .to(cursor, {
        opacity: 0,
        duration: 0.3,
        delay: 0.5 // Shorter delay before glitch
      })
      // Phase 2: Glitch animation
      .call(() => {
        // Remove cursor and set up for glitch
        cursor.remove()
        element.textContent = text
      })
      .to(element, {
        duration: 0.1,
        repeat: 8,
        yoyo: true,
        ease: "power2.inOut",
        textShadow: "2px 0 #ff0000, -2px 0 #00ff00",
        filter: "blur(1px)",
        onUpdate: function() {
          if (Math.random() < 0.5) {
            const glitchChars = "!@#$%^&*(){}[]|\\:;<>?/~`"
            let glitchText = ""
            for (let i = 0; i < text.length; i++) {
              glitchText += Math.random() < 0.1 
                ? glitchChars[Math.floor(Math.random() * glitchChars.length)]
                : text[i]
            }
            element.textContent = glitchText
          }
        }
      })
      .to(element, {
        textShadow: "none",
        filter: "blur(0px)",
        duration: 0.2,
        onComplete: () => {
          element.textContent = text
        }
      })

    // Cleanup function
    return () => {
      tl.kill()
      if (element) {
        gsap.set(element, { clearProps: "all" })
      }
    }
  }, [text, delay])

  return (
    <div ref={containerRef} className="inline-block">
      <span ref={textRef} className={className} />
    </div>
  )
}

export const HeroSection = () => {
  const sectionRef = useRef<HTMLElement>(null)

  useEffect(() => {
    const section = sectionRef.current
    if (!section) return

    // Create entrance animation for the entire section
    const tl = gsap.timeline()
    
    // Initial state
    gsap.set(section, {
      opacity: 0,
      y: 50
    })

    // Animate section entrance
    tl.to(section, {
      opacity: 1,
      y: 0,
      duration: 1,
      ease: "power3.out"
    })

    return () => {
      tl.kill()
    }
  }, [])

  return (
    <>
      <FloatingParticles />
      <section 
        ref={sectionRef}
        id="hero" 
        className="container py-32 pb-48 mx-auto relative z-10"
      >
        <div className="flex flex-col gap-4 px-4 ml-6 sm:ml-12 md:ml-16 lg:ml-20 xl:ml-24">
          <h1 className="text-2xl sm:text-2xl tracking-tight font-bold" role="banner">
            <AdvancedAnimatedText 
              text="Hi, I'm Kevin 👋" 
              animationType="reveal"
              delay={0.5}
            />
          </h1>
          <p className="max-w-[600px] text-muted-foreground text-md sm:text-lg tracking-tight" role="doc-subtitle">
            <SequentialAnimatedText 
              text="Software Engineer based in San Francisco" 
              delay={1}
            />
          </p>
          


          {/* Skip animation button for accessibility */}
          <button 
            className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 bg-primary text-primary-foreground px-4 py-2 rounded-md text-sm z-50"
            onClick={() => {
              // Skip all animations
              gsap.globalTimeline.progress(1)
              gsap.set('*', { clearProps: 'all' })
            }}
          >
            Skip animations
          </button>
        </div>

        {/* Performance optimizations and animations */}
        <style>{`
          @keyframes blink {
            0%, 50% { opacity: 1; }
            51%, 100% { opacity: 0; }
          }
          
          /* Hardware acceleration and performance hints */
          #hero {
            transform: translate3d(0, 0, 0);
            backface-visibility: hidden;
            perspective: 1000px;
          }
          
          /* Optimize text rendering for animations */
          #hero h1, #hero p {
            text-rendering: optimizeSpeed;
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
          }
          
          /* Respect reduced motion preferences */
          @media (prefers-reduced-motion: reduce) {
            #hero *, #hero *::before, #hero *::after {
              animation-duration: 0.01ms !important;
              animation-iteration-count: 1 !important;
              transition-duration: 0.01ms !important;
            }
          }
          
          /* High contrast mode support */
          @media (prefers-contrast: high) {
            #hero .text-muted-foreground {
              opacity: 1;
              font-weight: 600;
            }
          }
        `}</style>
      </section>
    </>
  )
}
