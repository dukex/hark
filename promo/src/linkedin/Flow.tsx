import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  interpolateColors,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Logo } from "../Logo";
import {
  ClaudeDesktop,
  MonitorEvent,
  ToolLine,
  UserMessage,
} from "../mockups/ClaudeDesktop";
import { GitHubPR } from "../mockups/providers";
import { Typed } from "../Typed";
import { Burst, GlowFilter, gridBackground, useHitGlow, Wire } from "./Circuit";

const clamp = {
  extrapolateLeft: "clamp",
  extrapolateRight: "clamp",
} as const;

const line = "pr.changes_requested acme/checkout#15 @carol";
const reply =
  "Carol wants the migration in its own PR. I can move it to a new branch and open that PR. Want me to?";

export const Flow = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const glow = useHitGlow([2.3]);

  return (
    <AbsoluteFill style={gridBackground}>
      <Interactive.Div
        name="Camera"
        style={{
          position: "absolute",
          inset: 0,
          translate: interpolate(
            frame,
            [4.4 * fps, 5.4 * fps],
            ["0px 0px", "0px -560px"],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.65, 0, 0.35, 1),
            },
          ),
        }}
      >
        <svg
          width={1080}
          height={900}
          style={{ position: "absolute", left: 0, top: 0 }}
        >
          <GlowFilter />
          <Wire
            d="M 540 500 L 540 600"
            color="#E6EDF3"
            pulses={[1.6]}
            duration={0.7}
          />
          <Wire
            d="M 540 740 L 540 840"
            color="#F2C94C"
            pulses={[3.1]}
            duration={0.7}
          />
        </svg>

        <div style={{ position: "absolute", left: 80, top: 100 }}>
          <GitHubPR
            width={920}
            height={400}
            review={interpolate(frame, [0.5 * fps, 1.1 * fps], [0, 1], {
              ...clamp,
              easing: Easing.bezier(0.16, 1, 0.3, 1),
            })}
          />
        </div>

        <div
          style={{
            position: "absolute",
            left: 80,
            top: 600,
            width: 920,
            height: 140,
            display: "flex",
            flexDirection: "column",
            justifyContent: "center",
            gap: 14,
            padding: "0 28px",
            borderRadius: 12,
            backgroundColor: "#0D0D0D",
            boxShadow: `0 0 ${12 + 48 * glow}px rgba(242, 201, 76, ${0.1 + 0.6 * glow})`,
            border: "2px solid",
            borderColor: interpolateColors(
              frame,
              [2.3 * fps, 2.6 * fps, 3.4 * fps],
              ["#333333", "#F2C94C", "#333333"],
            ),
            fontFamily: "JetBrains Mono",
          }}
        >
          <span
            style={{
              display: "flex",
              alignItems: "center",
              gap: 14,
              fontSize: 28,
              color: "#9E9E9E",
            }}
          >
            <Logo size={32} />$ hark github
          </span>
          <span style={{ fontSize: 30, color: "#FFFFFF" }}>
            <Typed
              text={line}
              count={Math.round(
                interpolate(
                  frame,
                  [2.3 * fps, 2.9 * fps],
                  [0, line.length],
                  clamp,
                ),
              )}
            />
          </span>
        </div>

        <div style={{ position: "absolute", left: 80, top: 840 }}>
          <ClaudeDesktop width={920} height={1050}>
            <UserMessage>
              Move the refund flow to the new payments client
            </UserMessage>
            <ToolLine name="Update" target="refund.go" stat="+18 −6" />
            <ToolLine name="Update" target="refund_test.go" stat="+42" />
            <MonitorEvent
              command="hark github"
              line={line}
              style={{
                opacity: interpolate(
                  frame,
                  [3.8 * fps, 4.1 * fps],
                  [0, 1],
                  clamp,
                ),
              }}
            />
            <span
              style={{
                marginTop: 12,
                fontSize: 44,
                lineHeight: 1.3,
                opacity: interpolate(
                  frame,
                  [5.4 * fps, 5.6 * fps],
                  [0, 1],
                  clamp,
                ),
              }}
            >
              <Typed
                text={reply}
                count={Math.round(
                  interpolate(
                    frame,
                    [5.6 * fps, 8.6 * fps],
                    [0, reply.length],
                    clamp,
                  ),
                )}
              />
            </span>
          </ClaudeDesktop>
        </div>

        <svg
          width={1080}
          height={900}
          style={{ position: "absolute", left: 0, top: 0 }}
        >
          <GlowFilter />
          <Burst x={540} y={600} at={2.3} color="#E6EDF3" />
          <Burst x={540} y={840} at={3.8} color="#F2C94C" />
        </svg>
      </Interactive.Div>
    </AbsoluteFill>
  );
};
