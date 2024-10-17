'use client'

export const HeroSection = () => {

  return (
    <section id="hero" className="container py-32 pb-48 mx-auto">
      <div className="flex flex-col gap-0">
          <h1 className={'text-2xl tracking-wide'}>
            Software Engineer in San Francisco
          </h1>
          <p className="max-w-[600px] text-muted-foreground text-md">
            I build and design software for the web, cloud, and beyond
          </p>
      </div>
    </section>
  )
}
