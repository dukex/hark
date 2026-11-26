import type { ReactNode } from "react";
import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import {
  DatadogMonitor,
  DiscordChannel,
  GitHubPR,
  JiraIssue,
  LinearInbox,
  SlackChannel,
  TodoistToday,
} from "../mockups/providers";

const Pop = ({
  left,
  top,
  at,
  children,
}: {
  readonly left: number;
  readonly top: number;
  readonly at: number;
  readonly children: ReactNode;
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const open = interpolate(frame, [at * fps, (at + 0.3) * fps], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.bezier(0.16, 1, 0.3, 1),
  });

  return (
    <div
      style={{
        position: "absolute",
        left,
        top,
        opacity: open,
        scale: 0.85 + 0.15 * open,
      }}
    >
      {children}
    </div>
  );
};

export const Tabs = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill>
      <Pop left={60} top={330} at={0.2}>
        <GitHubPR width={600} height={420} />
      </Pop>
      <Pop left={430} top={390} at={0.65}>
        <SlackChannel width={600} height={300} />
      </Pop>
      <Pop left={110} top={560} at={1.1}>
        <JiraIssue width={560} height={340} />
      </Pop>
      <Pop left={460} top={640} at={1.55}>
        <DatadogMonitor width={560} height={320} />
      </Pop>
      <Pop left={50} top={800} at={2}>
        <LinearInbox width={560} height={330} />
      </Pop>
      <Pop left={440} top={880} at={2.45}>
        <DiscordChannel width={580} height={260} />
      </Pop>
      <Pop left={220} top={960} at={2.9}>
        <TodoistToday width={560} height={300} />
      </Pop>
      <svg
        width={44}
        height={44}
        viewBox="0 0 24 24"
        style={{
          position: "absolute",
          left: interpolate(
            frame,
            [1 * fps, 1.6 * fps, 2.2 * fps, 2.8 * fps, 3.4 * fps, 4 * fps, 4.6 * fps, 5.2 * fps, 5.8 * fps],
            [300, 820, 360, 760, 300, 720, 480, 340, 860],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.65, 0, 0.35, 1),
            },
          ),
          top: interpolate(
            frame,
            [1 * fps, 1.6 * fps, 2.2 * fps, 2.8 * fps, 3.4 * fps, 4 * fps, 4.6 * fps, 5.2 * fps, 5.8 * fps],
            [480, 480, 680, 760, 900, 980, 1080, 440, 700],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.65, 0, 0.35, 1),
            },
          ),
          opacity: interpolate(frame, [0.8 * fps, 1 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        <path
          d="M4 2l16 10-7 1.5L9.5 21z"
          fill="#FFFFFF"
          stroke="#000000"
          strokeWidth={1.2}
          strokeLinejoin="round"
        />
      </svg>
      <AbsoluteFill
        style={{
          padding: "100px 80px 0",
          gap: 16,
          fontFamily: "Inter",
          backgroundImage:
            "linear-gradient(#0D0D0D 0px, #0D0D0D 300px, rgba(13, 13, 13, 0) 340px)",
        }}
      >
        <Interactive.Div
          name="Headline"
          style={{
            fontWeight: 700,
            fontSize: 88,
            letterSpacing: "-0.03em",
            color: "#FFFFFF",
            opacity: interpolate(frame, [1.2 * fps, 1.7 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          Seven tabs to watch
        </Interactive.Div>
        <Interactive.Div
          name="Subhead"
          style={{
            fontSize: 44,
            lineHeight: 1.25,
            color: "#9E9E9E",
            opacity: interpolate(frame, [2.6 * fps, 3.1 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          and every check pulls you out of the code.
        </Interactive.Div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
