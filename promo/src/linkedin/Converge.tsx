import {
  AbsoluteFill,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Logo } from "../Logo";
import { ClaudeDesktop } from "../mockups/ClaudeDesktop";

const services = [
  { name: "GitHub", color: "#24292F", left: 80, top: 330, event: "hark github · ci.failed" },
  { name: "Slack", color: "#4A154B", left: 310, top: 330, event: "hark slack · slack.dm" },
  { name: "Discord", color: "#5865F2", left: 540, top: 330, event: "hark discord · discord.mentioned" },
  { name: "Jira", color: "#0C66E4", left: 770, top: 330, event: "hark jira · jira.assigned" },
  { name: "Linear", color: "#5E6AD2", left: 195, top: 414, event: "hark linear · linear.assigned" },
  { name: "Datadog", color: "#632CA6", left: 425, top: 414, event: "hark datadog · datadog.no_data" },
  { name: "Todoist", color: "#DC4C3E", left: 655, top: 414, event: "hark todoist · task.due" },
];

const clamp = {
  extrapolateLeft: "clamp",
  extrapolateRight: "clamp",
} as const;

export const Converge = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const startOf = (index: number) => (0.5 + index * 0.45) * fps;

  return (
    <AbsoluteFill style={{ fontFamily: "Inter" }}>
      <Interactive.Div
        name="Headline"
        style={{
          position: "absolute",
          left: 80,
          right: 80,
          top: 100,
          fontWeight: 700,
          fontSize: 84,
          lineHeight: 1.05,
          letterSpacing: "-0.03em",
          color: "#FFFFFF",
          textWrap: "balance",
        }}
      >
        Your agent hears them all
      </Interactive.Div>

      <svg width={1080} height={1350} style={{ position: "absolute" }}>
        {services.map(({ name, left, top }) => (
          <line
            key={name}
            x1={left + 105}
            y1={top + 64}
            x2={540}
            y2={560}
            stroke="#333333"
            strokeWidth={3}
            strokeDasharray="10 8"
          />
        ))}
        <line
          x1={540}
          y1={670}
          x2={540}
          y2={760}
          stroke="#333333"
          strokeWidth={3}
          strokeDasharray="10 8"
        />
      </svg>

      {services.map(({ name, color, left, top }) => (
        <div
          key={name}
          style={{
            position: "absolute",
            left,
            top,
            width: 210,
            height: 64,
            display: "flex",
            alignItems: "center",
            gap: 14,
            padding: "0 16px",
            borderRadius: 14,
            backgroundColor: "#1A1A1A",
            border: "2px solid #333333",
            fontSize: 26,
            fontWeight: 600,
            color: "#FFFFFF",
          }}
        >
          <span
            style={{
              width: 32,
              height: 32,
              borderRadius: 8,
              backgroundColor: color,
            }}
          />
          {name}
        </div>
      ))}

      <div
        style={{
          position: "absolute",
          left: 390,
          top: 560,
          width: 300,
          height: 110,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: 18,
          borderRadius: 16,
          backgroundColor: "#1A1A1A",
          border: "2px solid #F2C94C",
          fontSize: 56,
          fontWeight: 700,
          color: "#FFFFFF",
        }}
      >
        <Logo size={64} />
        hark
      </div>

      <div style={{ position: "absolute", left: 80, top: 760 }}>
        <ClaudeDesktop width={920} height={540}>
          {services.map(({ name, event }, index) => (
            <span
              key={name}
              style={{
                marginTop: index === 0 ? 0 : -12,
                padding: "6px 14px",
                borderRadius: 8,
                borderLeft: "4px solid #D97757",
                backgroundColor: "#1F1E1D",
                fontFamily: "JetBrains Mono",
                fontSize: 19,
                opacity: interpolate(
                  frame,
                  [startOf(index) + 1.2 * fps, startOf(index) + 1.4 * fps],
                  [0, 1],
                  clamp,
                ),
              }}
            >
              Monitor · {event}
            </span>
          ))}
        </ClaudeDesktop>
      </div>

      {services.map(({ name, left, top }, index) => (
        <div key={name}>
          <div
            style={{
              position: "absolute",
              width: 20,
              height: 20,
              borderRadius: 9999,
              backgroundColor: "#FFFFFF",
              boxShadow: "0 0 20px 6px rgba(255, 255, 255, 0.45)",
              left: interpolate(
                frame,
                [startOf(index), startOf(index) + 0.7 * fps],
                [left + 95, 530],
                clamp,
              ),
              top: interpolate(
                frame,
                [startOf(index), startOf(index) + 0.7 * fps],
                [top + 54, 550],
                clamp,
              ),
              opacity: interpolate(
                frame,
                [
                  startOf(index),
                  startOf(index) + 0.1 * fps,
                  startOf(index) + 0.6 * fps,
                  startOf(index) + 0.7 * fps,
                ],
                [0, 1, 1, 0],
                clamp,
              ),
            }}
          />
          <div
            style={{
              position: "absolute",
              left: 530,
              width: 20,
              height: 20,
              borderRadius: 9999,
              backgroundColor: "#FFFFFF",
              boxShadow: "0 0 20px 6px rgba(255, 255, 255, 0.45)",
              top: interpolate(
                frame,
                [startOf(index) + 0.8 * fps, startOf(index) + 1.2 * fps],
                [660, 750],
                clamp,
              ),
              opacity: interpolate(
                frame,
                [
                  startOf(index) + 0.8 * fps,
                  startOf(index) + 0.9 * fps,
                  startOf(index) + 1.1 * fps,
                  startOf(index) + 1.2 * fps,
                ],
                [0, 1, 1, 0],
                clamp,
              ),
            }}
          />
        </div>
      ))}
    </AbsoluteFill>
  );
};
