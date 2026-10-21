import {
  IconBell,
  IconCamera,
  IconHome,
  IconMicrophone2,
  IconPlug,
  IconSettings,
  IconSun,
  IconVolume,
  IconWorld,
  type Icon,
} from "@tabler/icons-react";
import type { Domain, Field } from "./api";

export type Entry = { domain: string; field: Field };
export type Group = { title: string; entries: Entry[] };
export type Section = { key: string; slug: string; title: string; icon: Icon; groups: Group[]; count: number };

const icons: Record<string, Icon> = {
  "settings.network": IconWorld,
  "settings.features": IconPlug,
  "settings.display": IconSun,
  "home.title": IconHome,
  "settings.volume": IconVolume,
  "alerts.page": IconBell,
  "settings.camera": IconCamera,
  "voice.title": IconMicrophone2,
  "settings.system": IconSettings,
};

export function groupID(slug: string, index: number): string {
  return `g-${slug}-${index}`;
}

function slugOf(key: string): string {
  return key.split(".").find((p) => p !== "settings" && p !== "title" && p !== "page") ?? key;
}

export function sections(domains: Domain[]): Section[] {
  const out: Section[] = [];
  const byKey = new Map<string, Section>();
  for (const d of domains) {
    let s = byKey.get(d.section_key);
    if (!s) {
      s = {
        key: d.section_key,
        slug: slugOf(d.section_key),
        title: d.section,
        icon: icons[d.section_key] ?? IconSettings,
        groups: [],
        count: 0,
      };
      byKey.set(d.section_key, s);
      out.push(s);
    }
    for (const field of d.fields) {
      const title = field.group || d.domain;
      let g = s.groups.find((x) => x.title === title);
      if (!g) {
        g = { title, entries: [] };
        s.groups.push(g);
      }
      g.entries.push({ domain: d.domain, field });
      s.count++;
    }
  }
  return out.filter((s) => s.count > 0);
}
