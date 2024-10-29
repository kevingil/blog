// Hero.tsx
import styles from './Hero.module.css';

export const HeroSection = () => {
  return (
    <section id="hero" className={styles.hero}>
      <div className={styles.content}>
        <p className={styles.title}>
          Software Engineer in San Francisco
        </p>
        <p className={styles.description}>
          I build and design software for the web, cloud, and beyond
        </p>
      </div>
    </section>
  );
}
