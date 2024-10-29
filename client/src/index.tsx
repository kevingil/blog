
import { HeroSection } from "@/components/home/Hero";
import ArticlesList from '@/components/blog/ArticleList';
import { Container } from "@mantine/core";

export const metadata = {
  title: "Kevin Gil",
  description: "Software Engineer in San Francisco.",
  openGraph: {
    type: "website",
    url: "https://kevingil.com",
    title: "Kevin Gil",
    description: "Software Engineer in San Francisco.",
    images: [
      {
        url: "",
        width: 1200,
        height: 630,
        alt: "Kevin Gil",
      },
    ],
  },
};


export default function HomePage() {
  return (
    <Container size="lg" className="page">
      <HeroSection />
      <ArticlesList
        articles={[]}
        pagination={false} />
    </Container>
  );
}
