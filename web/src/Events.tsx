import { useMemo, useState } from "react";
import { ActionIcon, Code, Group, Paper, ScrollArea, Stack, Table, Text, TextInput, Tooltip } from "@mantine/core";
import { IconPlayerPause, IconPlayerPlay, IconSearch, IconTrash } from "@tabler/icons-react";

export type Seen = { at: Date; id: string; state: Record<string, unknown> };

type Props = {
  seen: Seen[];
  paused: boolean;
  setPaused: (p: boolean) => void;
  clear: () => void;
};

function summary(state: Record<string, unknown>): string {
  if ("state" in state && Object.keys(state).filter((k) => k !== "missing_state").length === 1) {
    return JSON.stringify(state.state);
  }
  return JSON.stringify(state);
}

export function Events({ seen, paused, setPaused, clear }: Props) {
  const [filter, setFilter] = useState("");
  const shown = useMemo(() => {
    const f = filter.trim().toLowerCase();
    return f ? seen.filter((s) => s.id.includes(f) || summary(s.state).toLowerCase().includes(f)) : seen;
  }, [seen, filter]);

  return (
    <Stack gap="sm" h="100%">
      <Group justify="space-between">
        <Text size="sm" c="dimmed">
          {shown.length} of {seen.length}
        </Text>
        <Group gap="xs">
          <TextInput
            size="xs"
            placeholder="Filter"
            leftSection={<IconSearch size={14} />}
            value={filter}
            onChange={(e) => setFilter(e.currentTarget.value)}
          />
          <Tooltip label={paused ? "Resume" : "Pause"}>
            <ActionIcon variant="default" onClick={() => setPaused(!paused)} aria-label={paused ? "Resume" : "Pause"}>
              {paused ? <IconPlayerPlay size={16} /> : <IconPlayerPause size={16} />}
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Clear">
            <ActionIcon variant="default" onClick={clear} aria-label="Clear">
              <IconTrash size={16} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Group>
      <Paper withBorder style={{ flex: 1, minHeight: 0 }}>
        <ScrollArea h="100%">
          <Table striped highlightOnHover verticalSpacing={6} fz="sm">
            <Table.Tbody>
              {shown.map((s, i) => (
                <Table.Tr key={`${s.at.getTime()}-${i}`}>
                  <Table.Td w={96}>
                    <Text size="xs" ff="monospace" c="dimmed">
                      {s.at.toLocaleTimeString([], { hour12: false })}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" ff="monospace" style={{ wordBreak: "break-all" }}>
                      {s.id}
                    </Text>
                    <Code style={{ wordBreak: "break-all" }}>{summary(s.state)}</Code>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
          {shown.length === 0 && (
            <Text c="dimmed" size="sm" p="md">
              Nothing yet
            </Text>
          )}
        </ScrollArea>
      </Paper>
    </Stack>
  );
}
