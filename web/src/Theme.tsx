import { ActionIcon, Menu, useMantineColorScheme, type MantineColorScheme } from "@mantine/core";
import { IconCheck, IconDeviceDesktop, IconMoon, IconSun, type Icon } from "@tabler/icons-react";

const schemes: { value: MantineColorScheme; label: string; icon: Icon }[] = [
  { value: "light", label: "Light", icon: IconSun },
  { value: "dark", label: "Dark", icon: IconMoon },
  { value: "auto", label: "Auto", icon: IconDeviceDesktop },
];

export function Theme() {
  const { colorScheme, setColorScheme } = useMantineColorScheme();
  const current = schemes.find((s) => s.value === colorScheme) ?? schemes[2];
  return (
    <Menu position="bottom-end" width={180}>
      <Menu.Target>
        <ActionIcon variant="default" size={32} aria-label="Theme">
          <current.icon size={16} />
        </ActionIcon>
      </Menu.Target>
      <Menu.Dropdown>
        {schemes.map((s) => (
          <Menu.Item
            key={s.value}
            leftSection={<s.icon size={14} />}
            rightSection={s.value === colorScheme ? <IconCheck size={14} /> : null}
            onClick={() => setColorScheme(s.value)}
          >
            {s.label}
          </Menu.Item>
        ))}
      </Menu.Dropdown>
    </Menu>
  );
}
