import { useTheme } from "@/components/theme-provider";
import { Button } from "../ui/button";
import { Moon, Sun } from "lucide-react";
import { cn } from "@/lib/utils";

type ToggleThemeProps = {
  onClick?: () => void;
  className?: string;
  showLabel?: boolean;
};

export function ToggleTheme({ onClick, className, showLabel = false }: ToggleThemeProps) {
  const { theme, setTheme } = useTheme();
  const isLight = theme === "light";

  return (
    <Button
      onClick={() => {
        setTheme(isLight ? "dark" : "light")
        if (onClick) {
          onClick()
        }
      }}
      size={showLabel ? "sm" : "icon"}
      variant="ghost"
      className={cn(
        showLabel ? "w-full justify-start" : "size-9 shrink-0 rounded-full",
        "text-foreground/70 hover:bg-accent hover:text-foreground",
        className
      )}
      aria-label="Toggle theme"
      title={`Switch to ${isLight ? "dark" : "light"} mode`}
    >
      {isLight ? (
        <Moon className="size-5" />
      ) : (
        <Sun className="size-5" />
      )}
      {showLabel && (
        <span>{isLight ? "Dark" : "Light"}</span>
      )}

      <span className="sr-only">Toggle theme</span>
    </Button>
  );
};
