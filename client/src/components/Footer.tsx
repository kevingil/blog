import { Text, Container } from '@mantine/core';
import styles from './Footer.module.css';

const data = [
  {
    title: 'Socials', 
    links: [
      { label: 'Github', link: 'https://github.com/kevingil' }, 
      { label: 'LinkedIn', link: 'https://linkedin.com/in/kevingil' }, 
      { label: 'Threads', link: 'https://www.threads.net/@kvngil' }, 
    ],
  },
  {
    title: 'Navigate', 
    links: [
      { label: 'Blog', link: '/blog' }, 
      { label: 'Contact', link: '/contact' }, 
      { label: 'About', link: '/about' },
    ],
  },
];


export function FooterSection() {
  const groups = data.map((group) => {
    const links = group.links.map((link, index) => (
      <Text<'a'>
        key={index}
        className={styles.link}
        component="a"
        href={link.link}
        onClick={(event) => event.preventDefault()}
      >
        {link.label}
      </Text>
    ));

    return (
      <div className={styles.wrapper} key={group.title}>
        <Text className={styles.title}>{group.title}</Text>
        {links}
      </div>
    );
  });

  return (
    <footer className={styles.footer}>
      <Container className={styles.inner}>
        <div className={styles.logo}>
          <Text  c="dimmed" className={styles.description}>
            Kevin Gil
          </Text>
        </div>
        <div className={styles.groups}>{groups}</div>
      </Container>
      <Container className={styles.afterFooter}>
        <Text size="xs" c="dimmed">
          © MIT License
        </Text>
      </Container>
    </footer>
  );
}
