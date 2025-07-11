import { useTheme } from "@/components/theme-provider";
import { Button } from "../ui/button";
import { Moon, Sun } from "lucide-react";

type ToggleThemeProps = {
  onClick?: () => void;
};

export function ToggleTheme({ onClick }: ToggleThemeProps) {
  const { theme, setTheme } = useTheme();
  return (
    <Button
      onClick={() => {
        setTheme(theme === "light" ? "dark" : "light")
        if (onClick) {
          onClick()
        }
      }}
      size="sm"
      variant="ghost"
      className="w-full justify-start"
    >
      <div className="flex gap-2 dark:hidden">
        <Moon className="size-5" />
        <span className="block lg:hidden"> Dark </span>
      </div>

      <div className="dark:flex gap-2 hidden">
        <Sun className="size-5" />
        <span className="block lg:hidden">Light</span>
      </div>

      <span className="sr-only">Toggle theme</span>
    </Button>
  );
};
