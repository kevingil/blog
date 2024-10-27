import React, { useEffect, useState } from 'react';
import { Skeleton, Card, Text, Anchor, Group } from '@mantine/core';
import { useDispatch, useSelector } from 'react-redux';
import axios from 'axios';
import { RootState } from '@/store'; 


interface ContactPage {
  id: number;
  title: string;
  content: string;
  emailAddress: string;
  socialLinks?: string;
  metaDescription?: string;
  lastUpdated: string;
}

const fetchContactPage = () => async (dispatch: any) => {
  dispatch({ type: 'CONTACT_PAGE_REQUEST' });
  try {
    const response = await axios.get<ContactPage>('/api/v1/contact'); 
    dispatch({ type: 'CONTACT_PAGE_SUCCESS', payload: response.data });
  } catch (error) {
    dispatch({ type: 'CONTACT_PAGE_FAILURE', error: error instanceof Error ? error.message : 'Unknown error' });
  }
};

const selectContactPage = (state: RootState) => state.contactPage;

export default function ContactPage() {
  const dispatch = useDispatch();
  const { pageData, isLoading, error } = useSelector(selectContactPage);
  const [socialLinks, setSocialLinks] = useState<Record<string, string>>({});

  useEffect(() => {
    dispatch(fetchContactPage());
  }, [dispatch]);

  useEffect(() => {
    if (pageData && pageData.socialLinks) {
      setSocialLinks(JSON.parse(pageData.socialLinks));
    }
  }, [pageData]);

  if (isLoading) {
    return (
      <div style={{ padding: '20px' }}>
        <Skeleton height={50} width={200} mb="md" />
        <Skeleton height={400} />
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: '20px' }}>
        <Text size="xl">Contact</Text>
        <Text>Failed to load page content.</Text>
      </div>
    );
  }

  if (!pageData) {
    return (
      <div style={{ padding: '20px' }}>
        <Text size="xl">Contact</Text>
        <Text>Page data is unavailable.</Text>
      </div>
    );
  }

  return (
    <div style={{ padding: '20px' }}>
      <Text size="xl" weight={500}>{pageData.title}</Text>
      
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
        <Card>
          <Card.Section>
            <Text>
              {pageData.content.split('\n').map((paragraph, index) => (
                <Text key={index} mb="sm">{paragraph}</Text>
              ))}
            </Text>
          </Card.Section>
        </Card>

        <Card>
          <Card.Section>
            <Text size="lg" >Email</Text>
            <Anchor href={`mailto:${pageData.emailAddress}`}>
              {pageData.emailAddress}
            </Anchor>
          </Card.Section>

          {Object.keys(socialLinks).length > 0 && (
            <Card.Section>
              <Text size="lg">Social Media</Text>
              <Group >
                {Object.entries(socialLinks).map(([platform, url]) => (
                  <Anchor
                    key={platform}
                    href={url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {platform.charAt(0).toUpperCase() + platform.slice(1)}
                  </Anchor>
                ))}
              </Group>
            </Card.Section>
          )}
        </Card>
      </div>

      <Text size="sm" color="gray" mt="md" style={{ display: 'none' }}>
        Last updated: {new Date(pageData.lastUpdated).toLocaleDateString()}
      </Text>
    </div>
  );
};

