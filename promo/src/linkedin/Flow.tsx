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

  return (
    <AbsoluteFill>
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
          <line
            x1={540}
            y1={500}
            x2={540}
            y2={600}
            stroke="#444444"
            strokeWidth={4}
            strokeDasharray="10 8"
          />
          <line
            x1={540}
            y1={740}
            x2={540}
            y2={840}
            stroke="#444444"
            strokeWidth={4}
            strokeDasharray="10 8"
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
              fontSize: 24,
              color: "#9E9E9E",
            }}
          >
            <Logo size={32} />$ hark github
          </span>
          <span style={{ fontSize: 28, color: "#FFFFFF" }}>
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

        <div
          style={{
            position: "absolute",
            left: 528,
            width: 24,
            height: 24,
            borderRadius: 9999,
            backgroundColor: "#FFFFFF",
            boxShadow: "0 0 24px 8px rgba(255, 255, 255, 0.45)",
            top: interpolate(
              frame,
              [1.6 * fps, 2.3 * fps, 3.1 * fps, 3.8 * fps],
              [490, 590, 730, 830],
              clamp,
            ),
            opacity: interpolate(
              frame,
              [
                1.6 * fps,
                1.7 * fps,
                2.2 * fps,
                2.3 * fps,
                3.1 * fps,
                3.2 * fps,
                3.7 * fps,
                3.8 * fps,
              ],
              [0, 1, 1, 0, 0, 1, 1, 0],
              clamp,
            ),
          }}
        />
      </Interactive.Div>
    </AbsoluteFill>
  );
};
