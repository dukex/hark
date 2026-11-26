import {
  AbsoluteFill,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Logo } from "../Logo";

const events = [
  { provider: "GitHub", line: "hark github · ci.failed" },
  { provider: "Slack", line: "hark slack · slack.dm" },
  { provider: "Discord", line: "hark discord · discord.mentioned" },
  { provider: "Jira", line: "hark jira · jira.assigned" },
  { provider: "Linear", line: "hark linear · linear.assigned" },
  { provider: "Datadog", line: "hark datadog · datadog.no_data" },
  { provider: "Todoist", line: "hark todoist · task.due" },
];

const clamp = {
  extrapolateLeft: "clamp",
  extrapolateRight: "clamp",
} as const;

const centerOf = (index: number) => 188 + index * 114;

export const Converge = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const startOf = (index: number) => (0.8 + index * 0.55) * fps;

  return (
    <AbsoluteFill style={{ fontFamily: "Inter" }}>
      <svg width={1920} height={1080} style={{ position: "absolute" }}>
        {events.map(({ provider }, index) => (
          <line
            key={provider}
            x1={440}
            y1={centerOf(index)}
            x2={820}
            y2={540}
            stroke="#333333"
            strokeWidth={3}
            strokeDasharray="12 10"
          />
        ))}
        <line
          x1={1140}
          y1={540}
          x2={1260}
          y2={540}
          stroke="#333333"
          strokeWidth={3}
          strokeDasharray="12 10"
        />
      </svg>

      {events.map(({ provider }, index) => (
        <div
          key={provider}
          style={{
            position: "absolute",
            left: 140,
            top: centerOf(index) - 38,
            width: 300,
            height: 76,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            borderRadius: 9999,
            backgroundColor: "#1A1A1A",
            border: "2px solid #333333",
            fontSize: 36,
            fontWeight: 600,
            color: "#FFFFFF",
          }}
        >
          {provider}
        </div>
      ))}

      <div
        style={{
          position: "absolute",
          left: 820,
          top: 470,
          width: 320,
          height: 140,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: 20,
          borderRadius: 16,
          backgroundColor: "#1A1A1A",
          border: "3px solid #333333",
          fontSize: 64,
          fontWeight: 700,
          color: "#FFFFFF",
        }}
      >
        <Logo size={72} />
        hark
      </div>

      <div
        style={{
          position: "absolute",
          left: 1260,
          top: 230,
          width: 540,
          height: 620,
          display: "flex",
          flexDirection: "column",
          backgroundColor: "#1A1A1A",
          border: "3px solid #333333",
          borderRadius: 16,
          overflow: "hidden",
        }}
      >
        <div
          style={{
            height: 56,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            borderBottom: "2px solid #333333",
            fontSize: 26,
            fontWeight: 600,
            color: "#9E9E9E",
          }}
        >
          Claude
        </div>
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 12,
            padding: 20,
            fontFamily: "JetBrains Mono",
            fontSize: 22,
            color: "#FFFFFF",
          }}
        >
          {events.map(({ provider, line }, index) => (
            <span
              key={provider}
              style={{
                padding: "10px 14px",
                borderRadius: 10,
                border: "2px solid #333333",
                backgroundColor: "#262626",
                opacity: interpolate(
                  frame,
                  [startOf(index) + 1.6 * fps, startOf(index) + 1.8 * fps],
                  [0, 1],
                  clamp,
                ),
              }}
            >
              {line}
            </span>
          ))}
        </div>
      </div>

      {events.map(({ provider }, index) => (
        <div key={provider}>
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
                [startOf(index), startOf(index) + 0.9 * fps],
                [430, 810],
                clamp,
              ),
              top: interpolate(
                frame,
                [startOf(index), startOf(index) + 0.9 * fps],
                [centerOf(index) - 10, 530],
                clamp,
              ),
              opacity: interpolate(
                frame,
                [
                  startOf(index),
                  startOf(index) + 0.1 * fps,
                  startOf(index) + 0.8 * fps,
                  startOf(index) + 0.9 * fps,
                ],
                [0, 1, 1, 0],
                clamp,
              ),
            }}
          />
          <div
            style={{
              position: "absolute",
              top: 530,
              width: 20,
              height: 20,
              borderRadius: 9999,
              backgroundColor: "#FFFFFF",
              boxShadow: "0 0 20px 6px rgba(255, 255, 255, 0.45)",
              left: interpolate(
                frame,
                [startOf(index) + 1 * fps, startOf(index) + 1.6 * fps],
                [1130, 1250],
                clamp,
              ),
              opacity: interpolate(
                frame,
                [
                  startOf(index) + 1 * fps,
                  startOf(index) + 1.1 * fps,
                  startOf(index) + 1.5 * fps,
                  startOf(index) + 1.6 * fps,
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
