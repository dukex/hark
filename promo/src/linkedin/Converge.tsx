import {
  AbsoluteFill,
  Interactive,
  interpolate,
  interpolateColors,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Logo } from "../Logo";
import { ClaudeDesktop } from "../mockups/ClaudeDesktop";
import { Burst, GlowFilter, gridBackground, useHitGlow, Wire } from "./Circuit";

const services = [
  { name: "GitHub", color: "#24292F", glow: "#E6EDF3", left: 80, top: 330, event: "hark github · ci.failed" },
  { name: "Slack", color: "#4A154B", glow: "#E01E5A", left: 310, top: 330, event: "hark slack · slack.dm" },
  { name: "Discord", color: "#5865F2", glow: "#7983F5", left: 540, top: 330, event: "hark discord · discord.mentioned" },
  { name: "Jira", color: "#0C66E4", glow: "#579DFF", left: 770, top: 330, event: "hark jira · jira.assigned" },
  { name: "Linear", color: "#5E6AD2", glow: "#8B93FF", left: 195, top: 414, event: "hark linear · linear.assigned" },
  { name: "Datadog", color: "#632CA6", glow: "#A77EF0", left: 425, top: 414, event: "hark datadog · datadog.no_data" },
  { name: "Todoist", color: "#DC4C3E", glow: "#E44332", left: 655, top: 414, event: "hark todoist · task.due" },
];

const startOf = (index: number) => 0.5 + index * 0.45;
const arrivals = services.map((_, index) => startOf(index) + 0.7);
const clamp = {
  extrapolateLeft: "clamp",
  extrapolateRight: "clamp",
} as const;

export const Converge = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const glow = useHitGlow(arrivals);

  return (
    <AbsoluteFill style={{ fontFamily: "Inter", ...gridBackground }}>
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
        <GlowFilter />
        {services.map(({ name, glow: color, left, top }, index) => (
          <Wire
            key={name}
            d={`M ${left + 105} ${top + 64} C ${left + 105} ${top + 150}, 540 ${470}, 540 560`}
            color={color}
            pulses={[startOf(index)]}
            duration={0.7}
          />
        ))}
        <Wire
          d="M 540 670 L 540 740"
          color="#F2C94C"
          pulses={arrivals.map((at) => at + 0.1)}
          duration={0.4}
        />
      </svg>

      {services.map(({ name, color, glow: flash, left, top }, index) => (
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
            border: "2px solid",
            borderColor: interpolateColors(
              frame,
              [
                startOf(index) * fps,
                (startOf(index) + 0.1) * fps,
                (startOf(index) + 0.7) * fps,
              ],
              ["#333333", flash, "#333333"],
            ),
            boxShadow: `0 0 ${interpolate(
              frame,
              [
                startOf(index) * fps,
                (startOf(index) + 0.1) * fps,
                (startOf(index) + 0.7) * fps,
              ],
              [0, 28, 0],
              clamp,
            )}px ${flash}`,
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
          boxShadow: `0 0 ${16 + 48 * glow}px rgba(242, 201, 76, ${0.2 + 0.6 * glow})`,
          scale: 1 + 0.04 * glow,
          fontSize: 56,
          fontWeight: 700,
          color: "#FFFFFF",
        }}
      >
        <Logo size={64} />
        hark
      </div>

      <div style={{ position: "absolute", left: 80, top: 740 }}>
        <ClaudeDesktop width={920} height={580}>
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
                fontSize: 26,
                opacity: interpolate(
                  frame,
                  [(arrivals[index] + 0.5) * fps, (arrivals[index] + 0.7) * fps],
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

      <svg
        width={1080}
        height={1350}
        style={{ position: "absolute", pointerEvents: "none" }}
      >
        <GlowFilter />
        {services.map(({ name, glow: color }, index) => (
          <Burst key={name} x={540} y={560} at={arrivals[index]} color={color} />
        ))}
        {arrivals.map((at) => (
          <Burst key={at} x={540} y={740} at={at + 0.5} color="#F2C94C" />
        ))}
      </svg>
    </AbsoluteFill>
  );
};
