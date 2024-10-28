import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import {
  Menu,
  Button,
  Stack,
  Text,
  Box,
  Paper,
  Group,
  Container,
  Anchor
} from '@mantine/core';
import { Users, Settings, Shield, PenLine, EllipsisVertical, ImageUp } from 'lucide-react';

export default function DashboardLayout() {
  const [isOpen, setIsOpen] = useState(false);
  const [currentPath, setCurrentPath] = useState('/dashboard');

  const navItems = [
    { href: '/dashboard', icon: Users, label: 'Profile' },
    { href: '/dashboard/blog', icon: PenLine, label: 'Articles' },
    { href: '/dashboard/uploads', icon: ImageUp, label: 'Uploads' },
    { href: '/dashboard/general', icon: Settings, label: 'General' },
    { href: '/dashboard/security', icon: Shield, label: 'Security' },
  ];

  const NavContent = ({ mobile = false }) => (
    <Stack gap={mobile ? "xs" : "sm"}>
      {navItems.map((item) => {
        const Icon = item.icon;
        return (
          <Anchor
            key={item.href}
            onClick={(e) => {
              e.preventDefault();
              setCurrentPath(item.href);
              setIsOpen(false);
            }}
            style={{ textDecoration: 'none' }}
          >
            {mobile ? (
              <Menu.Item
                leftSection={<Icon size={16} />}
                onClick={() => setIsOpen(false)}
              >
                {item.label}
              </Menu.Item>
            ) : (
              <Button
                variant={currentPath === item.href ? 'light' : 'subtle'}
                fullWidth
                leftSection={<Icon size={16} />}
                onClick={() => setIsOpen(false)}
                h={48}
                styles={(theme) => ({
                  root: {
                    borderRadius: theme.radius.md,
                    backgroundColor: currentPath === item.href
                      ? theme.primaryColor === 'dark'
                        ? theme.colors.dark[6]
                        : theme.colors.gray[1]
                      : 'transparent',
                    justifyContent: 'flex-start',
                  }
                })}
              >
                {item.label}
              </Button>
            )}
          </Anchor>
        );
      })}
    </Stack>
  );

  return (
    <Container size="xl" p={0}>
      <Box mih="80vh">
        {/* Mobile header */}
        <Box hiddenFrom="md" mb="xl">
          <Paper p="md" radius="md">
            <Group justify="space-between">
              <Text fw={500}>Dashboard</Text>
              <Menu
                opened={isOpen}
                onChange={setIsOpen}
                position="bottom-end"
                width={220}
              >
                <Menu.Target>
                  <Button variant="subtle" p={0} w={40} h={40}>
                    <EllipsisVertical size={24} />
                  </Button>
                </Menu.Target>
                <Menu.Dropdown>
                  <NavContent mobile />
                </Menu.Dropdown>
              </Menu>
            </Group>
          </Paper>
        </Box>

        <Group align="flex-start" gap={0}>
          {/* Desktop Sidebar */}
          <Box visibleFrom="md" w={256}>
            <Paper p="md" radius="md">
              <NavContent />
            </Paper>
          </Box>

          {/* Main content */}
          <Box style={{ flex: 1 }}>
            <Outlet />
          </Box>
        </Group>
      </Box>
    </Container>
  );
}
