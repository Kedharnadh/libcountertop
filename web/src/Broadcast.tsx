import { useCallback, useEffect, useRef, useState } from "react";
import { Badge, Button, Group, Progress, Stack, Text, ThemeIcon, Title, UnstyledButton } from "@mantine/core";
import { IconMicrophone, IconPlayerStopFilled, IconSpeakerphone } from "@tabler/icons-react";
import { api, SignedOut, type Device } from "./api";
import { notifications } from "./notify";

type Target = { id: string; name: string; model: string };
type Info = { targets: Target[]; rate: number; longest: number };

const shortest = 0.4;

export function Broadcast({ device, onSignedOut }: { device: Device; onSignedOut: (e: unknown) => void }) {
  const [info, setInfo] = useState<Info>();
  const [live, setLive] = useState(false);
  const [level, setLevel] = useState(0);
  const [length, setLength] = useState(0);
  const [result, setResult] = useState<string>();
  const [sending, setSending] = useState(false);
  const parts = useRef<Float32Array[]>([]);
  const stop = useRef<(() => void) | undefined>(undefined);

  const fail = useCallback(
    (e: Error) => (e instanceof SignedOut ? onSignedOut(e) : notifications.error(e.message)),
    [onSignedOut],
  );

  useEffect(() => {
    fetch("/api/v1/broadcast/", { credentials: "same-origin" })
      .then((r) => (r.ok ? (r.json() as Promise<Info>) : Promise.reject(new Error(r.statusText))))
      .then(setInfo)
      .catch(fail);
  }, [fail]);

  const send = useCallback(
    async (rate: number) => {
      const total = parts.current.reduce((n, p) => n + p.length, 0);
      if (total < rate * shortest) {
        setResult("Too short");
        return;
      }
      const pcm = new Int16Array(total);
      let at = 0;
      for (const p of parts.current) {
        for (let i = 0; i < p.length; i++) pcm[at++] = Math.max(-32768, Math.min(32767, Math.round(p[i] * 32767)));
      }
      setSending(true);
      try {
        const r = await fetch("/api/v1/broadcast/send", {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": `audio/L16; rate=${rate}; channels=1` },
          body: pcm.buffer,
        });
        if (r.status === 401) throw new SignedOut();
        if (!r.ok) throw new Error((await r.text()).trim() || r.statusText);
        const { sent } = (await r.json()) as { sent: number };
        setResult(sent ? `Played here and on ${sent} other device${sent === 1 ? "" : "s"}` : "Played here");
      } catch (e) {
        fail(e as Error);
      } finally {
        setSending(false);
      }
    },
    [fail],
  );

  const begin = useCallback(async () => {
    if (!info || live) return;
    setResult(undefined);
    parts.current = [];
    setLength(0);
    let stream: MediaStream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true },
      });
    } catch (e) {
      fail(e as Error);
      return;
    }
    const ctx = new AudioContext({ sampleRate: info.rate });
    const source = ctx.createMediaStreamSource(stream);
    const tap = ctx.createScriptProcessor(2048, 1, 1);
    let samples = 0;
    const most = info.rate * info.longest;
    tap.onaudioprocess = (e) => {
      const chunk = new Float32Array(e.inputBuffer.getChannelData(0));
      parts.current.push(chunk);
      samples += chunk.length;
      let sum = 0;
      for (let i = 0; i < chunk.length; i++) sum += chunk[i] * chunk[i];
      setLevel(Math.min(1, Math.sqrt(sum / chunk.length) * 4));
      setLength(samples / info.rate);
      if (samples >= most) stop.current?.();
    };
    source.connect(tap);
    tap.connect(ctx.destination);
    setLive(true);
    stop.current = () => {
      stop.current = undefined;
      tap.disconnect();
      source.disconnect();
      for (const t of stream.getTracks()) t.stop();
      void ctx.close();
      setLive(false);
      setLevel(0);
      void send(info.rate);
    };
  }, [fail, info, live, send]);

  useEffect(() => () => stop.current?.(), []);

  const targets = info?.targets ?? [];
  return (
    <Stack gap="lg" maw={760}>
      <Group gap="sm">
        <ThemeIcon variant="light" size="lg" radius="md">
          <IconSpeakerphone size={20} stroke={1.6} />
        </ThemeIcon>
        <Title order={3}>Broadcast</Title>
      </Group>

      <div className="ct-callcard">
        <Group gap={6} justify="center">
          <Badge variant="light" color="teal" tt="none">
            {device.name}
          </Badge>
          {targets.map((t) => (
            <Badge key={t.id} variant="light" color="gray" tt="none">
              {t.name}
            </Badge>
          ))}
        </Group>
        <UnstyledButton
          className="ct-mic"
          data-live={live || undefined}
          onClick={() => (live ? stop.current?.() : void begin())}
          aria-label={live ? "Stop and send" : "Start recording"}
          disabled={!info || sending}
        >
          {live && <span className="ct-mic-level" style={{ transform: `scale(${1 + level * 0.35})` }} />}
          {live ? <IconPlayerStopFilled size={44} /> : <IconMicrophone size={52} stroke={1.5} />}
        </UnstyledButton>
        <Stack gap={4} w="100%" maw={320} align="center">
          <Text fw={600}>{live ? "Recording" : sending ? "Sending…" : "Tap to talk"}</Text>
          {live && info && <Progress value={(length / info.longest) * 100} w="100%" size="sm" color="red" />}
          <Text size="sm" c="dimmed" ff="monospace">
            {live ? `${length.toFixed(1)} s` : result ?? `Up to ${info?.longest ?? 30} s`}
          </Text>
        </Stack>
        <Button variant="subtle" size="xs" onClick={() => api.recordBroadcast().then(() => setResult(`Speak to ${device.name} after the tone`)).catch(fail)}>
          Record on {device.name} instead
        </Button>
      </div>
    </Stack>
  );
}
