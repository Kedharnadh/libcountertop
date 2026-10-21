import { useMemo } from "react";
import { Group, Kbd, Text, UnstyledButton } from "@mantine/core";
import { Spotlight, spotlight, type SpotlightActionGroupData } from "@mantine/spotlight";
import { IconCode, IconGauge, IconLayoutDashboard, IconSearch, type Icon } from "@tabler/icons-react";
import type { Description, State } from "./api";
import { readable, shown } from "./entities";
import type { Section } from "./sections";

type Props = {
  sections: Section[];
  entities: Description[];
  states: Map<string, State["state"]>;
  apps: { slug: string; label: string; icon: Icon }[];
  go: (slug: string, focus?: string) => void;
};

export function Search({ sections, entities, states, apps, go }: Props) {
  const actions = useMemo<SpotlightActionGroupData[]>(() => {
    const pages = [
      {
        id: "p-overview",
        label: "Overview",
        leftSection: <IconLayoutDashboard size={18} stroke={1.6} />,
        onClick: () => go("overview"),
      },
      ...apps.map((a) => ({
        id: `p-${a.slug}`,
        label: a.label,
        leftSection: <a.icon size={18} stroke={1.6} />,
        onClick: () => go(a.slug),
      })),
      ...sections.map((s) => ({
        id: `p-${s.slug}`,
        label: s.title,
        leftSection: <s.icon size={18} stroke={1.6} />,
        onClick: () => go(s.slug),
      })),
      {
        id: "p-developer",
        label: "Developer",
        leftSection: <IconCode size={18} stroke={1.6} />,
        onClick: () => go("developer"),
      },
    ];
    const settings = sections.flatMap((s) =>
      s.groups.flatMap((g) =>
        g.entries.map(({ field }) => ({
          id: `f-${field.id}`,
          label: field.title,
          description: g.title === s.title ? s.title : `${s.title} › ${g.title}`,
          leftSection: <s.icon size={18} stroke={1.6} />,
          keywords: [field.id, field.name, g.title, s.title],
          onClick: () => go(s.slug, field.id),
        })),
      ),
    );
    const live = entities
      .filter((d) => readable.has(d.kind))
      .map((d) => {
        const unit = d.info.unit_of_measurement;
        const value = shown(d, states.get(d.id));
        return {
          id: `e-${d.id}`,
          label: d.info.name || d.id,
          description: unit && value !== "—" ? `${value} ${unit}` : value,
          leftSection: <IconGauge size={18} stroke={1.6} />,
          keywords: [d.id],
          onClick: () => go("developer", d.id),
        };
      });
    return [
      { group: "Pages", actions: pages },
      { group: "Settings", actions: settings },
      { group: "Readings", actions: live },
    ];
  }, [sections, entities, states, apps, go]);

  return (
    <Spotlight
      actions={actions}
      shortcut={["mod + K", "/"]}
      nothingFound="Nothing matches"
      highlightQuery
      limit={14}
      scrollable
      maxHeight={480}
      searchProps={{ leftSection: <IconSearch size={18} stroke={1.6} />, placeholder: "Search settings and readings" }}
    />
  );
}

export function SearchButton() {
  const mac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform);
  return (
    <UnstyledButton className="ct-search" onClick={() => spotlight.open()} aria-label="Search">
      <Group gap="xs" wrap="nowrap">
        <IconSearch size={15} stroke={1.8} />
        <Text size="sm" c="dimmed" visibleFrom="md">
          Search
        </Text>
        <Kbd size="xs" visibleFrom="md">
          {mac ? "⌘" : "Ctrl"} K
        </Kbd>
      </Group>
    </UnstyledButton>
  );
}
