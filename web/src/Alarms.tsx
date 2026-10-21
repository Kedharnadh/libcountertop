import { useCallback, useEffect, useState } from "react";
import {
  ActionIcon,
  Alert,
  Badge,
  Button,
  Chip,
  Group,
  Modal,
  Paper,
  SegmentedControl,
  Select,
  Slider,
  Stack,
  Switch,
  Text,
  TextInput,
  ThemeIcon,
  Title,
} from "@mantine/core";
import { IconAlarm, IconBellRinging, IconPencil, IconPlus, IconTrash } from "@tabler/icons-react";
import { api, SignedOut, type Alarm, type Alarms as State } from "./api";
import { notifications } from "./notify";

const week = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
const dayNames: Record<string, string> = {
  mon: "Mon",
  tue: "Tue",
  wed: "Wed",
  thu: "Thu",
  fri: "Fri",
  sat: "Sat",
  sun: "Sun",
};
const volumeNames: Record<string, string> = { static: "Full", gentle: "Rising", silent: "Silent" };

const fresh: Alarm = {
  id: "",
  name: "",
  time: "07:00",
  days: [],
  enabled: true,
  sound: "timer",
  snooze_minutes: 9,
  volume: "gentle",
  ramp_seconds: 60,
  light_minutes: 0,
};

function repeats(days: string[]): string {
  const set = new Set(days);
  if (set.size === 0) return "Once";
  if (set.size === 7) return "Every day";
  if (set.size === 5 && !set.has("sat") && !set.has("sun")) return "Weekdays";
  if (set.size === 2 && set.has("sat") && set.has("sun")) return "Weekends";
  return week.filter((d) => set.has(d)).map((d) => dayNames[d]).join(" ");
}

function clock(d: Date): string {
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hourCycle: "h23" });
}

function when(at: string): string {
  const d = new Date(at);
  const today = new Date();
  const tomorrow = new Date(today.getFullYear(), today.getMonth(), today.getDate() + 1);
  const day =
    d.toDateString() === today.toDateString()
      ? "today"
      : d.toDateString() === tomorrow.toDateString()
        ? "tomorrow"
        : d.toLocaleDateString([], { weekday: "long" });
  return `${day} at ${clock(d)}`;
}

export function Alarms({ onSignedOut }: { onSignedOut: (e: unknown) => void }) {
  const [state, setState] = useState<State>();
  const [editing, setEditing] = useState<Alarm>();
  const [removing, setRemoving] = useState<Alarm>();

  const load = useCallback(() => {
    api
      .alarms()
      .then(setState)
      .catch((e: Error) => (e instanceof SignedOut ? onSignedOut(e) : notifications.error(e.message)));
  }, [onSignedOut]);

  useEffect(() => {
    load();
    const every = window.setInterval(load, 15000);
    window.addEventListener("focus", load);
    return () => {
      window.clearInterval(every);
      window.removeEventListener("focus", load);
    };
  }, [load]);

  const run = (p: Promise<unknown>) =>
    p.then(load).catch((e: Error) => (e instanceof SignedOut ? onSignedOut(e) : notifications.error(e.message)));

  const list = state?.alarms ?? [];
  const ringing = list.find((a) => a.id === state?.ringing);
  const next = list.find((a) => a.id === state?.next);

  return (
    <Stack gap="lg" maw={760}>
      <Group justify="space-between">
        <Group gap="sm">
          <ThemeIcon variant="light" size="lg" radius="md">
            <IconAlarm size={20} stroke={1.6} />
          </ThemeIcon>
          <Title order={3}>Alarms</Title>
        </Group>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setEditing({ ...fresh })} disabled={!state}>
          Add alarm
        </Button>
      </Group>

      {ringing && (
        <Alert color="red" variant="light" icon={<IconBellRinging size={20} />} title={`${ringing.name || "Alarm"} is ringing`}>
          <Group gap="xs" mt="xs">
            <Button color="red" size="xs" onClick={() => run(api.stopAlarm())}>
              Stop
            </Button>
            <Button variant="default" size="xs" onClick={() => run(api.snoozeAlarm())}>
              Snooze
            </Button>
          </Group>
        </Alert>
      )}

      {next && state?.next_at && !ringing && (
        <Text size="sm" c="dimmed">
          Next: {next.name || next.time}, {when(state.next_at)}
        </Text>
      )}

      {state && list.length === 0 && (
        <Paper withBorder p="xl">
          <Text c="dimmed" ta="center">
            No alarms yet
          </Text>
        </Paper>
      )}

      <Stack gap="sm">
        {list.map((a) => {
          const snoozed = state?.snoozed?.[a.id];
          return (
            <Paper key={a.id} withBorder p="md" className="ct-alarm" data-off={!a.enabled || undefined}>
              <Group justify="space-between" wrap="nowrap" gap="md">
                <Stack gap={2} style={{ minWidth: 0 }}>
                  <Group gap="xs" align="baseline" wrap="nowrap">
                    <Text fz={32} fw={600} ff="monospace" lh={1.1} className="ct-alarm-time">
                      {a.time}
                    </Text>
                    {a.id === state?.next && !ringing && (
                      <Badge variant="light" size="sm">
                        Next
                      </Badge>
                    )}
                    {snoozed && (
                      <Badge variant="light" color="orange" size="sm">
                        Snoozed until {clock(new Date(snoozed * 1000))}
                      </Badge>
                    )}
                  </Group>
                  <Text size="sm" c="dimmed" truncate>
                    {[a.name, repeats(a.days), volumeNames[a.volume] ?? a.volume, a.light_minutes ? `${a.light_minutes} min light` : ""]
                      .filter(Boolean)
                      .join(" · ")}
                  </Text>
                </Stack>
                <Group gap="xs" wrap="nowrap">
                  <ActionIcon variant="subtle" color="gray" aria-label="Edit" onClick={() => setEditing({ ...a })}>
                    <IconPencil size={18} />
                  </ActionIcon>
                  <ActionIcon variant="subtle" color="gray" aria-label="Delete" onClick={() => setRemoving(a)}>
                    <IconTrash size={18} />
                  </ActionIcon>
                  <Switch
                    size="md"
                    checked={a.enabled}
                    aria-label="Enabled"
                    onChange={(e) => run(api.enableAlarm(a.id, e.currentTarget.checked))}
                  />
                </Group>
              </Group>
            </Paper>
          );
        })}
      </Stack>

      {editing && state && (
        <Editor
          alarm={editing}
          limits={state}
          onClose={() => setEditing(undefined)}
          onSave={(a) => run(api.saveAlarm(a).then(() => setEditing(undefined)))}
        />
      )}

      <Modal opened={!!removing} onClose={() => setRemoving(undefined)} title="Delete alarm" centered size="sm">
        <Text size="sm">
          Delete {removing?.name ? `“${removing.name}”` : "the alarm"} at {removing?.time}?
        </Text>
        <Group justify="flex-end" mt="lg">
          <Button variant="default" onClick={() => setRemoving(undefined)}>
            Cancel
          </Button>
          <Button
            color="red"
            onClick={() => removing && run(api.deleteAlarm(removing.id).then(() => setRemoving(undefined)))}
          >
            Delete
          </Button>
        </Group>
      </Modal>
    </Stack>
  );
}

type EditorProps = { alarm: Alarm; limits: State; onClose: () => void; onSave: (a: Alarm) => void };

function Editor({ alarm, limits, onClose, onSave }: EditorProps) {
  const [a, setA] = useState(alarm);
  const set = <K extends keyof Alarm>(k: K, v: Alarm[K]) => setA((x) => ({ ...x, [k]: v }));
  const sounds = limits.sounds ?? [];
  return (
    <Modal opened onClose={onClose} title={alarm.id ? "Edit alarm" : "New alarm"} centered size="md">
      <Stack gap="md">
        <Group grow align="flex-end">
          <TextInput
            label="Time"
            type="time"
            value={a.time}
            onChange={(e) => set("time", e.currentTarget.value)}
            styles={{ input: { fontFamily: "var(--mantine-font-family-monospace)", fontSize: 18 } }}
          />
          <TextInput label="Name" placeholder="Alarm" value={a.name} onChange={(e) => set("name", e.currentTarget.value)} />
        </Group>
        <Stack gap={6}>
          <Text size="sm" fw={500}>
            Repeat
          </Text>
          <Chip.Group multiple value={a.days} onChange={(v) => set("days", v)}>
            <Group gap={6}>
              {week.map((d) => (
                <Chip key={d} value={d} size="sm" variant="outline">
                  {dayNames[d]}
                </Chip>
              ))}
            </Group>
          </Chip.Group>
          <Text size="xs" c="dimmed">
            {repeats(a.days)}
          </Text>
        </Stack>
        <Select
          label="Sound"
          data={sounds}
          value={a.sound}
          onChange={(v) => v && set("sound", v)}
          allowDeselect={false}
          comboboxProps={{ withinPortal: true }}
        />
        <Stack gap={6}>
          <Text size="sm" fw={500}>
            Volume
          </Text>
          <SegmentedControl
            data={limits.volumes.map((v) => ({ value: v, label: volumeNames[v] ?? v }))}
            value={a.volume}
            onChange={(v) => setA((x) => ({ ...x, volume: v, ramp_seconds: Math.max(x.ramp_seconds, limits.ramp_least) }))}
          />
        </Stack>
        {a.volume === "gentle" && (
          <Level
            label="Rise over"
            value={a.ramp_seconds}
            min={limits.ramp_least}
            max={limits.ramp_most}
            step={15}
            show={(v) => (v < 60 ? `${v} s` : `${Math.floor(v / 60)} min${v % 60 ? ` ${v % 60} s` : ""}`)}
            onChange={(v) => set("ramp_seconds", v)}
          />
        )}
        <Level
          label="Snooze"
          value={a.snooze_minutes}
          min={0}
          max={limits.snooze_most}
          step={1}
          show={(v) => (v ? `${v} min` : "Off")}
          onChange={(v) => set("snooze_minutes", v)}
        />
        <Level
          label="Wake light"
          value={a.light_minutes}
          min={0}
          max={limits.light_most}
          step={1}
          show={(v) => (v ? `${v} min before` : "Off")}
          onChange={(v) => set("light_minutes", v)}
        />
        <Switch label="Enabled" checked={a.enabled} onChange={(e) => set("enabled", e.currentTarget.checked)} />
        <Group justify="flex-end" mt="sm">
          <Button variant="default" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={() => onSave(a)} disabled={!/^\d\d:\d\d$/.test(a.time)}>
            Save
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

type LevelProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step: number;
  show: (v: number) => string;
  onChange: (v: number) => void;
};

function Level({ label, value, min, max, step, show, onChange }: LevelProps) {
  return (
    <Stack gap={6}>
      <Group justify="space-between">
        <Text size="sm" fw={500}>
          {label}
        </Text>
        <Text size="sm" c="dimmed" ff="monospace">
          {show(value)}
        </Text>
      </Group>
      <Slider value={value} min={min} max={max} step={step} onChange={onChange} label={null} />
    </Stack>
  );
}
