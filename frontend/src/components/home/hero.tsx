'use client'
import { motion, Variants } from 'framer-motion'
import React from 'react'

const containerVariants: Variants = {
  hidden: {},
  visible: {
    transition: {
      staggerChildren: 0.015, 
    },
  },
}

const characterVariants: Variants = {
  hidden: {
    opacity: 0,
    scale: 3,
    y: 80, 
    filter: 'blur(2rem)',
  },
  visible: {
    opacity: 1,
    y: 0,
    scale: 1,
    filter: 'blur(0)',
    transition: {
      duration: 0.4,
      ease: 'easeOut',
    },
  },
}

// Blur and fade in from the bottom with delay
function AnimatedText({ text }: { text: string }) {
  const characters: string[] = [];
  
  for (const char of text) {
    characters.push(char);
  }
  
  return (
    <motion.div
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className="inline-flex flex-wrap"
    >
      {characters.map((char, index) => (
        <motion.span
          key={`${char}-${index}`}
          variants={characterVariants}
          className="whitespace-pre"
        >
          {char}
        </motion.span>
      ))}
    </motion.div>
  );
}

// Basic animate from bottom
function BasicAnimatedText({ text }: { text: string }) {
  const [visible, setVisible] = React.useState(false)

  React.useEffect(() => {
    setVisible(true)
  }, [])

  return (
    <span
      className={`hide-down ${
        visible ? 'animate-down' : ''
      }`}
    >
      {text}
    </span>
  )
}

export const HeroSection = () => {

  return (
    <section id="hero" className="container py-32 pb-48 mx-auto relative z-10">
      <div className="flex flex-col gap-0 px-4">
          <h1 className={'text-xl sm:text-2xl tracking-tight'}>
          <AnimatedText text="Hi, I'm Kevin 👋" />
          </h1>
          <p className="max-w-[600px] text-muted-foreground text-sm sm:text-md tracking-tight">
            <BasicAnimatedText text="Software Engineer based in San Francisco" />
          </p>
      </div>
    </section>
  )
}
