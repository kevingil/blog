import { 
  Grid, 
  Tooltip, 
  Text, 
  Group, 
  RingProgress, 
  Stack,
  Title,
  SimpleGrid,
  Card,
  ThemeIcon,
  Progress,
  Center,
  SegmentedControl
} from '@mantine/core';
import { 
  Users, 
  TrendingUp, 
  DollarSign, 
  Clock,
  ArrowUpRight,
  ArrowDownRight,
  Eye,
  FileText,
  MessageSquare
} from 'lucide-react';
import { useState } from 'react';

const statsData = [
  { icon: Eye, label: 'Total Views', value: '32.4K', increase: '+24%', color: 'blue' },
  { icon: FileText, label: 'Total Posts', value: '145', increase: '+12%', color: 'teal' },
  { icon: MessageSquare, label: 'Comments', value: '2,814', increase: '+18%', color: 'violet' },
];

export default function DashboardOverview() {
  const [timeRange, setTimeRange] = useState('7days');

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="center">
        <div>
          <Title order={2}>Dashboard Overview</Title>
          <Text c="dimmed" size="sm">Track your performance and metrics</Text>
        </div>
        <SegmentedControl
          value={timeRange}
          onChange={setTimeRange}
          data={[
            { label: '7 days', value: '7days' },
            { label: '30 days', value: '30days' },
            { label: '3 months', value: '3months' },
          ]}
        />
      </Group>

      {/* Stats Cards */}
      <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
        {statsData.map((stat) => (
          <Card key={stat.label} withBorder padding="lg">
            <Group justify="space-between">
              <ThemeIcon size="xl" color={stat.color} variant="light">
                <stat.icon size={20} />
              </ThemeIcon>
              <Text size="xs" c="dimmed" fw={700}>
                {stat.increase}
                {Number(stat.increase.replace('%', '')) > 0 ? (
                  <ArrowUpRight size={16} style={{ display: 'inline' }} />
                ) : (
                  <ArrowDownRight size={16} style={{ display: 'inline' }} />
                )}
              </Text>
            </Group>
            <Text size="xl" fw={700} mt="md">
              {stat.value}
            </Text>
            <Text size="sm" c="dimmed" mt={4}>
              {stat.label}
            </Text>
          </Card>
        ))}
      </SimpleGrid>

      <Grid>
        {/* Activity Overview */}
        <Grid.Col span={{ base: 12, md: 8 }}>
          <Card withBorder>
            <Group justify="space-between" mb="md">
              <Text fw={700}>Activity Overview</Text>
              <ThemeIcon variant="light" color="gray">
                <TrendingUp size={16} />
              </ThemeIcon>
            </Group>
            <Progress.Root size="xl">
              <Tooltip label="Comments">
                <Progress.Section value={35} color="violet" />
              </Tooltip>
              <Tooltip label="Posts">
                <Progress.Section value={28} color="teal" />
              </Tooltip>
              <Tooltip label="Views">
                <Progress.Section value={25} color="blue" />
              </Tooltip>
            </Progress.Root>
            <SimpleGrid cols={3} mt="md">
              <div>
                <Text c="dimmed" size="xs">Comments</Text>
                <Text fw={700}>35%</Text>
              </div>
              <div>
                <Text c="dimmed" size="xs">Posts</Text>
                <Text fw={700}>28%</Text>
              </div>
              <div>
                <Text c="dimmed" size="xs">Views</Text>
                <Text fw={700}>25%</Text>
              </div>
            </SimpleGrid>
          </Card>
        </Grid.Col>

        {/* User Stats */}
        <Grid.Col span={{ base: 12, md: 4 }}>
          <Card withBorder h="100%">
            <Stack justify="space-between" h="100%">
              <Group justify="space-between">
                <Text fw={700}>User Stats</Text>
                <ThemeIcon variant="light" color="blue">
                  <Users size={16} />
                </ThemeIcon>
              </Group>
              
              <Center>
                <RingProgress
                  size={180}
                  thickness={16}
                  roundCaps
                  sections={[
                    { value: 40, color: 'cyan', tooltip: 'New Users' },
                    { value: 25, color: 'orange', tooltip: 'Returning Users' },
                    { value: 15, color: 'grape', tooltip: 'Inactive Users' },
                  ]}
                  label={
                    <Center>
                      <Stack gap={0} align="center">
                        <Text size="xl" fw={700}>
                          80%
                        </Text>
                        <Text size="xs" c="dimmed">
                          Active Users
                        </Text>
                      </Stack>
                    </Center>
                  }
                />
              </Center>

              <SimpleGrid cols={3}>
                <Stack gap={0} align="center">
                  <Text fw={700}>2.1k</Text>
                  <Text size="xs" c="dimmed">New</Text>
                </Stack>
                <Stack gap={0} align="center">
                  <Text fw={700}>1.2k</Text>
                  <Text size="xs" c="dimmed">Returning</Text>
                </Stack>
                <Stack gap={0} align="center">
                  <Text fw={700}>0.8k</Text>
                  <Text size="xs" c="dimmed">Inactive</Text>
                </Stack>
              </SimpleGrid>
            </Stack>
          </Card>
        </Grid.Col>
      </Grid>
    </Stack>
  );
}
