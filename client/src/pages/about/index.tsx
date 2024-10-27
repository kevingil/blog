import React, { useEffect, useState } from 'react';
import { Card, Text, Skeleton, Image, Group } from '@mantine/core';
import { useDispatch, useSelector } from 'react-redux';
import axios from 'axios';
import { RootState } from '@/store'; 


interface AboutPage {
  id: number;
  title: string;
  content: string;
  profileImage?: string;
  metaDescription?: string;
  lastUpdated: string;
}


const fetchAboutPage = () => async (dispatch: any) => {
  dispatch({ type: 'ABOUT_PAGE_REQUEST' });
  try {
    const response = await axios.get<AboutPage>('/api/about'); 
    dispatch({ type: 'ABOUT_PAGE_SUCCESS', payload: response.data });
  } catch (error) {
    dispatch({ type: 'ABOUT_PAGE_FAILURE', error: error instanceof Error ? error.message : 'Unknown error' });
  }
};

const selectAboutPage = (state: RootState) => state.aboutPage;

export default function AboutPage() {
  const dispatch = useDispatch();
  const { pageData, isLoading, error } = useSelector(selectAboutPage);

  useEffect(() => {
    dispatch(fetchAboutPage());
  }, [dispatch]);

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
        <Text size="xl" >About</Text>
        <Text>Failed to load page content.</Text>
      </div>
    );
  }

  if (!pageData) {
    return (
      <div style={{ padding: '20px' }}>
        <Text size="xl" >About</Text>
        <Text>Page data is unavailable.</Text>
      </div>
    );
  }

  return (
    <div style={{ padding: '20px' }}>
      <Text size="xl" >{pageData.title}</Text>
      
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: '20px' }}>
        {pageData.profileImage && (
          <div>
            <Card>
              <Card.Section>
                <Image 
                  src={pageData.profileImage} 
                  alt="Profile"
                  radius="md"
                />
              </Card.Section>
            </Card>
          </div>
        )}
        
        <div>
          <Card>
            <Card.Section>
              <Text>
                {pageData.content.split('\n').map((paragraph, index) => (
                  <Text key={index} mb="sm">{paragraph}</Text>
                ))}
              </Text>
            </Card.Section>
          </Card>
        </div>
      </div>

      <Text size="sm" color="gray" mt="md" style={{ display: 'none' }}>
        Last updated: {new Date(pageData.lastUpdated).toLocaleDateString()}
      </Text>
    </div>
  );
};

