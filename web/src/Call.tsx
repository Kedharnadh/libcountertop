import { useCallback, useEffect, useRef, useState } from "react";
import { Group, SimpleGrid, Stack, Text, ThemeIcon, Title, Tooltip, UnstyledButton } from "@mantine/core";
import {
  IconMaximize,
  IconMicrophone,
  IconMicrophoneOff,
  IconPhone,
  IconPhoneCall,
  IconPhoneOff,
  IconVideo,
  IconVideoOff,
  type Icon,
} from "@tabler/icons-react";
import { api, SignedOut, type CallNow, type CallView, type Device, type Peer } from "./api";
import { notifications } from "./notify";

const viewEvery = 1500;
const gatherMost = 3000;

const ended: Record<string, string> = {
  hangup: "Call ended",
  declined: "Declined",
  busy: "Busy",
  unanswered: "No answer",
  refused: "Not taking calls",
  unavailable: "Not taking calls",
  failed: "Call failed",
  dropped: "Connection lost",
};

type Phase = "idle" | "calling" | "ringing" | "talking" | "ended";

function initials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  return ((words[0]?.[0] ?? "") + (words[1]?.[0] ?? "")).toUpperCase() || "?";
}

function clock(since: number, now: number): string {
  const s = Math.max(0, Math.floor((now - since) / 1000));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

function gathered(pc: RTCPeerConnection): Promise<void> {
  if (pc.iceGatheringState === "complete") return Promise.resolve();
  return new Promise((done) => {
    const timer = window.setTimeout(done, gatherMost);
    pc.addEventListener("icegatheringstatechange", () => {
      if (pc.iceGatheringState === "complete") {
        window.clearTimeout(timer);
        done();
      }
    });
  });
}

function preferH264(t: RTCRtpTransceiver) {
  const codecs = RTCRtpReceiver.getCapabilities("video")?.codecs ?? [];
  const h264 = codecs.filter((c) => c.mimeType === "video/H264");
  if (h264.length && t.setCodecPreferences) {
    t.setCodecPreferences([...h264, ...codecs.filter((c) => c.mimeType !== "video/H264")]);
  }
}

function useBrowserCall(deviceVideo: boolean, fail: (e: Error) => void) {
  const [phase, setPhase] = useState<Phase>("idle");
  const [reason, setReason] = useState("");
  const [video, setVideo] = useState(false);
  const [muted, setMuted] = useState(false);
  const [blind, setBlind] = useState(false);
  const [started, setStarted] = useState(0);
  const pc = useRef<RTCPeerConnection | undefined>(undefined);
  const local = useRef<MediaStream | undefined>(undefined);
  const remote = useRef<MediaStream | undefined>(undefined);
  const session = useRef<string | undefined>(undefined);
  const far = useRef<HTMLVideoElement | null>(null);
  const near = useRef<HTMLVideoElement | null>(null);

  const attach = useCallback(() => {
    if (far.current && remote.current && far.current.srcObject !== remote.current) far.current.srcObject = remote.current;
    if (near.current && local.current && near.current.srcObject !== local.current) near.current.srcObject = local.current;
  }, []);

  const release = useCallback(() => {
    pc.current?.close();
    pc.current = undefined;
    for (const t of local.current?.getTracks() ?? []) t.stop();
    local.current = undefined;
    remote.current = undefined;
    session.current = undefined;
  }, []);

  const finish = useCallback(
    (why: string) => {
      release();
      setReason(why);
      setPhase("ended");
    },
    [release],
  );

  const listen = useCallback(
    async (id: string) => {
      while (session.current === id) {
        let s;
        try {
          s = await api.signal(id);
        } catch (e) {
          if (e instanceof SignedOut) throw e;
          continue;
        }
        if (session.current !== id) return;
        if (!s) continue;
        if (s.type === "answer") {
          try {
            await pc.current?.setRemoteDescription({ type: "answer", sdp: s.sdp });
            setStarted(Date.now());
            setPhase("talking");
            attach();
          } catch {
            void api.endCall(id).catch(() => undefined);
            finish("failed");
            return;
          }
        } else {
          finish(s.reason || "hangup");
          return;
        }
      }
    },
    [attach, finish],
  );

  const start = useCallback(
    async (withVideo: boolean) => {
      setVideo(withVideo);
      setMuted(false);
      setBlind(false);
      setReason("");
      setPhase("calling");
      try {
        local.current = await navigator.mediaDevices.getUserMedia({ audio: true, video: withVideo });
        const conn = new RTCPeerConnection();
        pc.current = conn;
        remote.current = new MediaStream();
        conn.ontrack = (e) => {
          remote.current?.addTrack(e.track);
          attach();
        };
        conn.onconnectionstatechange = () => {
          if (conn.connectionState === "failed") finish("dropped");
        };
        for (const t of local.current.getTracks()) conn.addTrack(t, local.current);
        const sending = conn.getTransceivers().find((t) => t.sender.track?.kind === "video");
        if (sending) preferH264(sending);
        if (!withVideo && deviceVideo) conn.addTransceiver("video", { direction: "recvonly" });
        await conn.setLocalDescription(await conn.createOffer());
        await gathered(conn);
        attach();
        const size = local.current.getVideoTracks()[0]?.getSettings();
        const r = await api.offer({
          sdp: conn.localDescription?.sdp ?? "",
          video: withVideo,
          width: size?.width ?? 0,
          height: size?.height ?? 0,
        });
        if (!r.session) {
          finish(r.status);
          return;
        }
        session.current = r.session;
        setPhase("ringing");
        void listen(r.session).catch(fail);
      } catch (e) {
        if (e instanceof SignedOut) fail(e);
        finish("failed");
      }
    },
    [attach, deviceVideo, fail, finish, listen],
  );

  const hangUp = useCallback(() => {
    const id = session.current;
    if (id) void api.endCall(id).catch(() => undefined);
    finish("hangup");
  }, [finish]);

  const toggleMute = () => {
    const next = !muted;
    setMuted(next);
    for (const t of local.current?.getAudioTracks() ?? []) t.enabled = !next;
  };

  const toggleCamera = () => {
    const next = !blind;
    setBlind(next);
    for (const t of local.current?.getVideoTracks() ?? []) t.enabled = !next;
  };

  useEffect(() => () => {
    const id = session.current;
    if (id) void api.endCall(id).catch(() => undefined);
    release();
  }, [release]);

  useEffect(attach);

  return { phase, reason, video, muted, blind, started, far, near, start, hangUp, toggleMute, toggleCamera, reset: () => setPhase("idle") };
}

function Round({ icon: I, label, tone, on, onClick, big }: { icon: Icon; label: string; tone?: "go" | "stop"; on?: boolean; onClick: () => void; big?: boolean }) {
  return (
    <Tooltip label={label}>
      <UnstyledButton className="ct-round" data-tone={tone} data-on={on || undefined} data-big={big || undefined} onClick={onClick} aria-label={label}>
        <I size={big ? 28 : 22} stroke={1.7} />
      </UnstyledButton>
    </Tooltip>
  );
}

function Avatar({ name, board, pulse }: { name: string; board?: string; pulse?: boolean }) {
  if (board) {
    return (
      <div className="ct-device" data-pulse={pulse || undefined}>
        {pulse && (
          <>
            <span className="ct-ripple" />
            <span className="ct-ripple" style={{ animationDelay: "0.8s" }} />
          </>
        )}
        <img src={`/api/v1/devices/${encodeURIComponent(board)}`} alt={name} />
      </div>
    );
  }
  return (
    <div className="ct-avatar" data-pulse={pulse || undefined}>
      {pulse && (
        <>
          <span className="ct-ripple" />
          <span className="ct-ripple" style={{ animationDelay: "0.8s" }} />
        </>
      )}
      <span>{initials(name)}</span>
    </div>
  );
}

function useNow(active: boolean) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!active) return;
    const t = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(t);
  }, [active]);
  return now;
}

export function Call({ device, onSignedOut }: { device: Device; onSignedOut: (e: unknown) => void }) {
  const [view, setView] = useState<CallView>();
  const fail = useCallback(
    (e: Error) => (e instanceof SignedOut ? onSignedOut(e) : notifications.error(e.message)),
    [onSignedOut],
  );
  const browser = useBrowserCall(view?.video ?? false, fail);
  const now = useNow(true);

  const load = useCallback(() => {
    api.callView().then(setView).catch(fail);
  }, [fail]);

  useEffect(() => {
    load();
    const t = window.setInterval(load, viewEvery);
    return () => window.clearInterval(t);
  }, [load]);

  const act = (p: Promise<unknown>) => p.then(load).catch(fail);
  const call = view?.call;
  const browserLive = browser.phase === "calling" || browser.phase === "ringing" || browser.phase === "talking";

  return (
    <Stack gap="lg" maw={980}>
      <Group gap="sm">
        <ThemeIcon variant="light" size="lg" radius="md">
          <IconPhone size={20} stroke={1.6} />
        </ThemeIcon>
        <Title order={3}>Call</Title>
      </Group>

      {browserLive || browser.phase === "ended" ? (
        <BrowserStage device={device} b={browser} now={now} />
      ) : call ? (
        <DeviceStage device={device} call={call} now={now} act={act} />
      ) : null}

      {!browserLive && !call && (
        <SimpleGrid cols={{ base: 1, md: 2 }} spacing="lg">
          <div className="ct-callcard">
            <Text size="xs" fw={700} tt="uppercase" lts={0.6} c="dimmed">
              From this browser
            </Text>
            <Avatar name={device.name} board={device.board} />
            <Text fw={600} size="lg">
              {device.name}
            </Text>
            <Group gap="lg">
              <Round icon={IconPhone} label="Voice call" tone="go" big onClick={() => void browser.start(false)} />
              {view?.video && <Round icon={IconVideo} label="Video call" tone="go" big onClick={() => void browser.start(true)} />}
            </Group>
          </div>
          <div className="ct-callcard">
            <Text size="xs" fw={700} tt="uppercase" lts={0.6} c="dimmed">
              From {device.name}
            </Text>
            {view && view.peers.length === 0 && (
              <Text c="dimmed" size="sm" py="xl">
                No other devices in this home
              </Text>
            )}
            <div className="ct-peers">
              {view?.peers.map((p) => (
                <PeerTile key={p.id} peer={p} video={view.video} dial={(v) => act(api.dial(p.id, v))} />
              ))}
            </div>
          </div>
        </SimpleGrid>
      )}
    </Stack>
  );
}

function PeerTile({ peer, video, dial }: { peer: Peer; video: boolean; dial: (video: boolean) => void }) {
  return (
    <div className="ct-peer">
      <Avatar name={peer.name} board={peer.board} />
      <Text size="sm" fw={600} truncate maw="100%">
        {peer.name}
      </Text>
      <Text size="xs" c="dimmed" truncate maw="100%">
        {peer.model}
      </Text>
      <Group gap={6} className="ct-peer-actions">
        <Round icon={IconPhone} label={`Call ${peer.name}`} tone="go" onClick={() => dial(false)} />
        {video && peer.video && <Round icon={IconVideo} label={`Video call ${peer.name}`} tone="go" onClick={() => dial(true)} />}
      </Group>
    </div>
  );
}

function DeviceStage({ device, call, now, act }: { device: Device; call: CallNow; now: number; act: (p: Promise<unknown>) => void }) {
  const status =
    call.state === "talking"
      ? call.reconnecting
        ? "Reconnecting…"
        : clock(Date.parse(call.since), now)
      : call.state === "ringing"
        ? "Incoming call"
        : "Calling…";
  return (
    <div className="ct-stage">
      <Text size="xs" c="dimmed" tt="uppercase" fw={700} lts={0.6}>
        {device.name} {call.state === "ringing" ? "← " : "→ "}
        {call.browser ? "this browser" : call.peer.name}
      </Text>
      <Avatar name={call.peer.name} board={call.browser ? undefined : call.peer.board} pulse={call.state !== "talking"} />
      <Stack gap={2} align="center">
        <Text fw={600} size="xl">
          {call.peer.name}
        </Text>
        <Text c="dimmed" ff={call.state === "talking" ? "monospace" : undefined}>
          {status}
          {call.video ? " · video" : ""}
          {call.remote_muted ? " · they muted" : ""}
        </Text>
      </Stack>
      <Group gap="lg">
        {call.state === "ringing" && <Round icon={IconPhoneCall} label="Answer" tone="go" big onClick={() => act(api.answer())} />}
        {call.state === "talking" && (
          <Round icon={call.muted ? IconMicrophoneOff : IconMicrophone} label={call.muted ? "Unmute" : "Mute"} on={call.muted} onClick={() => act(api.muteCall(!call.muted))} />
        )}
        {call.state === "talking" && call.video && (
          <Round icon={call.camera_off ? IconVideoOff : IconVideo} label={call.camera_off ? "Camera on" : "Camera off"} on={call.camera_off} onClick={() => act(api.cameraCall(!call.camera_off))} />
        )}
        <Round icon={IconPhoneOff} label="Hang up" tone="stop" big onClick={() => act(api.hangup())} />
      </Group>
    </div>
  );
}

function BrowserStage({ device, b, now }: { device: Device; b: ReturnType<typeof useBrowserCall>; now: number }) {
  const stage = useRef<HTMLDivElement>(null);
  const status =
    b.phase === "talking" ? clock(b.started, now) : b.phase === "ringing" ? "Ringing…" : b.phase === "calling" ? "Calling…" : (ended[b.reason] ?? "Call ended");
  const live = b.phase !== "ended";
  return (
    <div className="ct-stage" data-video={(live && (b.video || b.phase === "talking")) || undefined} ref={stage}>
      {live && <video ref={b.far} autoPlay playsInline className="ct-far" />}
      {live && b.video && <video ref={b.near} autoPlay playsInline muted className="ct-near" />}
      <div className="ct-stage-top">
        <Avatar name={device.name} board={device.board} pulse={b.phase === "calling" || b.phase === "ringing"} />
        <Stack gap={2} align="center">
          <Text fw={600} size="xl">
            {device.name}
          </Text>
          <Text c="dimmed" ff={b.phase === "talking" ? "monospace" : undefined}>
            {status}
          </Text>
        </Stack>
      </div>
      <Group gap="lg" className="ct-stage-controls">
        {live ? (
          <>
            <Round icon={b.muted ? IconMicrophoneOff : IconMicrophone} label={b.muted ? "Unmute" : "Mute"} on={b.muted} onClick={b.toggleMute} />
            {b.video && <Round icon={b.blind ? IconVideoOff : IconVideo} label={b.blind ? "Camera on" : "Camera off"} on={b.blind} onClick={b.toggleCamera} />}
            {b.video && <Round icon={IconMaximize} label="Full screen" onClick={() => void stage.current?.requestFullscreen().catch(() => undefined)} />}
            <Round icon={IconPhoneOff} label="Hang up" tone="stop" big onClick={b.hangUp} />
          </>
        ) : (
          <Round icon={IconPhone} label="Back" onClick={b.reset} />
        )}
      </Group>
    </div>
  );
}
