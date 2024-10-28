import { Card, Text, Skeleton } from '@mantine/core';

export default function SettingsSkeleton() {
  return (
    <section style={{ flex: 1, padding: 0 }}>
      {/* Page title skeleton */}
      <Skeleton height={40} width={128} mb="md" />

      {/* About Page Card */}
      <Card mb="md">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '16px' }}>
          <Text>
            <Skeleton height={24} width={96} />
          </Text>
          <Skeleton height={32} width={32} radius="md" /> {/* Edit button skeleton */}
        </div>
        <div style={{ padding: '16px' }}>
          <div style={{ marginBottom: '16px' }}>
            {/* Title field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={48} /> {/* Label */}
              <Skeleton height={40} width="100%" /> {/* Content */}
            </div>
            
            {/* Content field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={64} /> {/* Label */}
              <Skeleton height={96} width="100%" /> {/* Content */}
            </div>
            
            {/* Profile Image URL field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={128} /> {/* Label */}
              <Skeleton height={40} width="100%" /> {/* Content */}
            </div>
            
            {/* Meta Description field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={112} /> {/* Label */}
              <Skeleton height={64} width="100%" /> {/* Content */}
            </div>
          </div>
        </div>
      </Card>

      {/* Contact Page Card */}
      <Card mb="md">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '16px' }}>
          <Text>
            <Skeleton height={24} width={112} />
          </Text>
          <Skeleton height={32} width={32} radius="md" /> {/* Edit button skeleton */}
        </div>
        <div style={{ padding: '16px' }}>
          <div style={{ marginBottom: '16px' }}>
            {/* Title field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={48} /> {/* Label */}
              <Skeleton height={40} width="100%" /> {/* Content */}
            </div>
            
            {/* Content field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={64} /> {/* Label */}
              <Skeleton height={96} width="100%" /> {/* Content */}
            </div>
            
            {/* Email Address field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={96} /> {/* Label */}
              <Skeleton height={40} width="100%" /> {/* Content */}
            </div>
            
            {/* Social Links field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={96} /> {/* Label */}
              <Skeleton height={80} width="100%" /> {/* Content */}
            </div>
            
            {/* Meta Description field */}
            <div style={{ marginBottom: '8px' }}>
              <Skeleton height={16} width={112} /> {/* Label */}
              <Skeleton height={64} width="100%" /> {/* Content */}
            </div>
          </div>
        </div>
      </Card>
    </section>
  );
}
