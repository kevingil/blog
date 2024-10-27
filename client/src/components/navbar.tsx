import React, { useState } from "react";
import { Link } from "react-router-dom";
import { Button, Avatar, Menu, Transition } from "@mantine/core";
import { Home, LogOut } from "lucide-react"; 
import classes from './Navbar.module.css'; 

const title = "Kevin Gil";

const routeList = [
  {
    href: "/blog",
    label: "Blog",
  },
  {
    href: "/contact",
    label: "Contact",
  },
  {
    href: "/about",
    label: "About",
  },
];

export const Navbar = () => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const user = null;

  return (
    <header className={classes.navbar}>
      <div className={classes.navbarWrapper}>

        <div className={classes.title}>
          <Link to="/">
          {title}
          </Link>
        </div>

        <nav className={classes.navLinks}>
          {routeList.map(({ href, label }) => (
            <Button
              key={href}
              component="a"
              href={href}
              variant="subtle"
              className={classes.link}
            >
              {label}
            </Button>
          ))}
          {user && (
            <div className={classes.userMenu}>

              <Menu
                opened={isMenuOpen}
                onOpen={() => setIsMenuOpen(true)}
                onClose={() => setIsMenuOpen(false)}
                trigger="click"
                position="bottom-end"
              >
                <Menu.Target>
                  <Avatar
                    className={classes.avatar}
                    onClick={() => setIsMenuOpen((prev) => !prev)}
                  >
                  </Avatar>
                </Menu.Target>

                <Transition transition="fade" duration={100} mounted={isMenuOpen}>
                  {(styles) => (
                    <Menu.Dropdown style={styles}>
                      <Menu.Item component="a" href="/dashboard">
                        <Home className={classes.icon} /> Dashboard
                      </Menu.Item>
                      <Menu.Item color="red">
                        <LogOut className={classes.icon} /> Sign out
                      </Menu.Item>
                    </Menu.Dropdown>
                  )}
                </Transition>
              </Menu>
            </div>
          )}
        </nav>



      </div>
    </header>
  );
};
