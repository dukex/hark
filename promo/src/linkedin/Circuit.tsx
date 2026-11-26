import { getLength, getPointAtLength } from "@remotion/paths";
import {
  Easing,
  interpolate,
  random,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";

const clamp = {
  extrapolateLeft: "clamp",
  extrapolateRight: "clamp",
} as const;

export const gridBackground = {
  backgroundColor: "#0D0D0D",
  backgroundImage:
    "radial-gradient(rgba(255, 255, 255, 0.07) 1.5px, transparent 1.5px)",
  backgroundSize: "28px 28px",
};

export const GlowFilter = () => (
  <defs>
    <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
      <feGaussianBlur stdDeviation="6" result="blur" />
      <feMerge>
        <feMergeNode in="blur" />
        <feMergeNode in="SourceGraphic" />
      </feMerge>
    </filter>
  </defs>
);

const Comet = ({
  d,
  length,
  color,
  at,
  duration,
}: {
  readonly d: string;
  readonly length: number;
  readonly color: string;
  readonly at: number;
  readonly duration: number;
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const tail = Math.min(140, length * 0.6);
  const travel = interpolate(
    frame,
    [at * fps, (at + (duration * (length + tail)) / length) * fps],
    [0, length + tail],
    clamp,
  );

  if (travel === 0 || travel === length + tail) {
    return null;
  }

  const head = getPointAtLength(d, Math.min(travel, length));

  return (
    <g>
      <path
        d={d}
        fill="none"
        stroke={color}
        strokeWidth={10}
        strokeLinecap="round"
        strokeDasharray={`${tail} ${length + tail}`}
        strokeDashoffset={tail - travel}
        filter="url(#glow)"
      />
      <path
        d={d}
        fill="none"
        stroke="#FFFFFF"
        strokeWidth={3}
        strokeLinecap="round"
        strokeDasharray={`${tail} ${length + tail}`}
        strokeDashoffset={tail - travel}
      />
      {head && travel < length ? (
        <circle cx={head.x} cy={head.y} r={7} fill="#FFFFFF" filter="url(#glow)" />
      ) : null}
    </g>
  );
};

export const Wire = ({
  d,
  color,
  pulses,
  duration,
}: {
  readonly d: string;
  readonly color: string;
  readonly pulses: readonly number[];
  readonly duration: number;
}) => {
  const frame = useCurrentFrame();
  const length = getLength(d);

  return (
    <g>
      <path d={d} fill="none" stroke="#262626" strokeWidth={2} />
      <path
        d={d}
        fill="none"
        stroke={color}
        strokeOpacity={0.4}
        strokeWidth={2}
        strokeDasharray="3 13"
        strokeDashoffset={-frame * 2.5}
      />
      {pulses.map((at) => (
        <Comet
          key={at}
          d={d}
          length={length}
          color={color}
          at={at}
          duration={duration}
        />
      ))}
    </g>
  );
};

export const Burst = ({
  x,
  y,
  at,
  color,
}: {
  readonly x: number;
  readonly y: number;
  readonly at: number;
  readonly color: string;
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = interpolate(frame, [at * fps, (at + 0.45) * fps], [0, 1], clamp);

  if (t === 0 || t === 1) {
    return null;
  }

  const spread = Easing.out(Easing.cubic)(t);

  return (
    <g opacity={1 - t} filter="url(#glow)">
      <circle
        cx={x}
        cy={y}
        r={8 + 56 * spread}
        fill="none"
        stroke={color}
        strokeWidth={3}
      />
      {Array.from({ length: 12 }, (_, index) => {
        const angle =
          (index / 12) * Math.PI * 2 + random(`spark-${at}-${index}`) * 0.4;
        const reach = 0.7 + random(`reach-${at}-${index}`) * 0.6;
        const inner = (12 + 60 * spread) * reach;
        const outer = inner + 22 * (1 - t);
        return (
          <line
            key={index}
            x1={x + Math.cos(angle) * inner}
            y1={y + Math.sin(angle) * inner}
            x2={x + Math.cos(angle) * outer}
            y2={y + Math.sin(angle) * outer}
            stroke={index % 3 === 0 ? "#FFFFFF" : color}
            strokeWidth={3}
            strokeLinecap="round"
          />
        );
      })}
    </g>
  );
};

export const useHitGlow = (hits: readonly number[]) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return Math.max(
    0,
    ...hits.map((at) =>
      interpolate(
        frame,
        [at * fps - 1, at * fps, (at + 0.5) * fps],
        [0, 1, 0],
        clamp,
      ),
    ),
  );
};
