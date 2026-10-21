import { useEffect, useRef, useState } from "react";
import { Indicator, Tooltip, UnstyledButton } from "@mantine/core";

type Props = { live: boolean; beat: number; unseen: number; onClick: () => void };

const restMs = 650;
const trace = "M1 12 H7 L9 9 L11 12 H12 L14 3 L16 20 L18 12 H20 L22 10 L24 12 H31";

export function Heart({ live, beat, unseen, onClick }: Props) {
  const [shown, setShown] = useState(0);
  const last = useRef(0);
  const queued = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (beat === 0) return;
    const wait = last.current + restMs - Date.now();
    if (wait <= 0) {
      last.current = Date.now();
      setShown((n) => n + 1);
      return;
    }
    if (queued.current === undefined) {
      queued.current = window.setTimeout(() => {
        queued.current = undefined;
        last.current = Date.now();
        setShown((n) => n + 1);
      }, wait);
    }
  }, [beat]);

  useEffect(() => () => window.clearTimeout(queued.current), []);

  const label = live ? `Events${unseen ? ` · ${unseen} new` : ""}` : "Reconnecting";
  return (
    <Tooltip label={label}>
      <Indicator
        label={unseen > 99 ? "99+" : unseen}
        disabled={!unseen}
        size={16}
        offset={4}
        color="red"
      >
        <UnstyledButton className="ct-heart" data-live={live || undefined} onClick={onClick} aria-label={label}>
          {shown > 0 && <span key={`r${shown}`} className="ct-heart-ring" />}
          <svg viewBox="0 0 32 24" width="26" height="20" aria-hidden>
            <path className="ct-heart-base" d={live ? trace : "M1 12 H31"} />
            {live && shown > 0 && <path key={shown} className="ct-heart-pulse" d={trace} pathLength={100} />}
          </svg>
        </UnstyledButton>
      </Indicator>
    </Tooltip>
  );
}
